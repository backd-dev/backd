// Deno Worker wrapper for executing user functions

const functionPath = Deno.env.get("BACKD_FUNCTION_PATH");
const internalUrl = Deno.env.get("BACKD_INTERNAL_URL");
const app = Deno.env.get("BACKD_APP");

if (!functionPath || !internalUrl || !app) {
  throw new Error("Missing required environment variables");
}

// Import the user function
let userFunction: any;
try {
  const module = await import(functionPath);
  userFunction = module.default || module.handler;
} catch (error) {
  throw new Error(`Failed to load function from ${functionPath}: ${error.message}`);
}

// Handle messages from the main thread
self.onmessage = async (event) => {
  const { method, headers, body, params } = event.data;
  
  try {
    // Create Request object
    const request = new Request(`${internalUrl}/functions/${app}`, {
      method: method || "POST",
      headers: headers || {},
      body: body || null,
    });

    // Create a simple backd client interface
    const backd = {
      query: async (query: string, params?: any[]) => {
        const response = await fetch(`${internalUrl}/internal/query`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ app, query, params }),
        });
        return response.json();
      },
      secret: async (name: string) => {
        const response = await fetch(`${internalUrl}/internal/secret`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ app, name }),
        });
        const data = await response.json();
        return data.secret;
      },
      auth: {
        validate: async (token: string) => {
          const response = await fetch(`${internalUrl}/internal/auth`, {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ app, token }),
          });
          return response.json();
        },
      },
      job: {
        enqueue: async (functionName: string, input: any, options?: any) => {
          const response = await fetch(`${internalUrl}/internal/jobs`, {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({
              app,
              function: functionName,
              input: JSON.stringify(input),
              trigger: "function",
              ...options,
            }),
          });
          return response.json();
        },
      },
    };

    // Call the user function
    let result;
    if (typeof userFunction === "function") {
      result = await userFunction(request, backd);
    } else {
      throw new Error("Function must export a default function or handler");
    }

    // Handle different response types
    let responseBody = "";
    let responseStatus = 200;
    let responseHeaders: Record<string, string> = {};

    if (result instanceof Response) {
      responseStatus = result.status;
      responseHeaders = Object.fromEntries(result.headers.entries());
      responseBody = await result.text();
    } else if (typeof result === "string") {
      responseBody = result;
    } else if (typeof result === "object") {
      responseBody = JSON.stringify(result);
      responseHeaders["Content-Type"] = "application/json";
    } else {
      responseBody = String(result);
    }

    // Send response back to main thread
    self.postMessage({
      status: responseStatus,
      headers: responseHeaders,
      body: responseBody,
    });

  } catch (error) {
    // Send error response
    self.postMessage({
      status: 500,
      headers: {},
      body: "",
      error: error.message,
    });
  } finally {
    // Close the worker
    self.close();
  }
};
