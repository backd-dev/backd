import { serve } from "https://deno.land/std@0.208.0/http/server.ts";

// Get environment variables
const socketPath = Deno.env.get("BACKD_SOCKET_PATH");
const internalUrl = Deno.env.get("BACKD_INTERNAL_URL");
const functionsRoot = Deno.env.get("BACKD_FUNCTIONS_ROOT");

if (!socketPath || !internalUrl || !functionsRoot) {
  console.error("Missing required environment variables");
  Deno.exit(1);
}

// Print READY signal to indicate we're ready
console.log("READY");

// Create Unix socket listener
const listener = await Deno.listen({
  path: socketPath,
  transport: "unix",
});

// Handle connections
for await (const conn of listener) {
  handleConnection(conn);
}

async function handleConnection(conn: Deno.Conn) {
  const buffer = new Uint8Array(1024);
  
  try {
    // Read request
    const { n, eof } = await conn.read(buffer);
    if (eof || n === 0) {
      conn.close();
      return;
    }

    const requestText = new TextDecoder().decode(buffer.subarray(0, n)).trim();
    const request = JSON.parse(requestText);

    // Handle the request
    const response = await handleRequest(request);
    
    // Send response
    const responseText = JSON.stringify(response) + "\n";
    await conn.write(new TextEncoder().encode(responseText));
    
  } catch (error) {
    // Send error response
    const errorResponse = {
      id: request?.id || "unknown",
      status: 500,
      headers: {},
      body: "",
      error: error.message,
    };
    
    const responseText = JSON.stringify(errorResponse) + "\n";
    await conn.write(new TextEncoder().encode(responseText));
  } finally {
    conn.close();
  }
}

async function handleRequest(request: any): Promise<any> {
  const { id, app, function: functionName, method, headers, body, params, timeout } = request;
  
  // Resolve function path
  const functionPath = `${functionsRoot}/${functionName}/index.ts`;
  
  try {
    // Check if function file exists
    await Deno.stat(functionPath);
  } catch (error) {
    return {
      id,
      status: 404,
      headers: {},
      body: "",
      error: `Function ${functionName} not found`,
    };
  }

  // Create worker to run the function
  const worker = new Worker(new URL("./worker_wrapper.ts", import.meta.url), {
    type: "module",
    env: {
      BACKD_FUNCTION_PATH: functionPath,
      BACKD_INTERNAL_URL: internalUrl,
      BACKD_APP: app,
    },
  });

  return new Promise((resolve, reject) => {
    // Set timeout
    const timeoutId = setTimeout(() => {
      worker.terminate();
      resolve({
        id,
        status: 408,
        headers: {},
        body: "",
        error: "Function execution timeout",
      });
    }, timeout || 30000);

    // Send message to worker
    worker.postMessage({
      method,
      headers,
      body,
      params,
    });

    // Handle response
    worker.onmessage = (event) => {
      clearTimeout(timeoutId);
      worker.terminate();
      
      const response = event.data;
      resolve({
        id,
        status: response.status || 200,
        headers: response.headers || {},
        body: response.body || "",
        error: response.error,
      });
    };

    // Handle errors
    worker.onerror = (error) => {
      clearTimeout(timeoutId);
      worker.terminate();
      
      resolve({
        id,
        status: 500,
        headers: {},
        body: "",
        error: error.message,
      });
    };
  });
}
