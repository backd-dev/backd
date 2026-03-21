/**
 * Backd Function Template
 * 
 * This is a template for creating backd functions.
 * Replace this content with your function implementation.
 */

// Function input/output types (example)
interface Input {
  name: string;
  message?: string;
}

interface Output {
  greeting: string;
  timestamp: string;
}

// Main function handler
export default async function handler(input: Input): Promise<Output> {
  // Access request context (if available)
  // Note: backd SDK will be available at runtime
  // const user = await backd.auth.getUser();
  // const app = await backd.auth.getApp();

  console.log(`Function called with input: ${JSON.stringify(input)}`);

  // Your function logic here
  const greeting = input.message 
    ? `${input.name}: ${input.message}`
    : `Hello, ${input.name}!`;

  return {
    greeting,
    timestamp: new Date().toISOString(),
  };
}

// Optional: Define function metadata
export const config = {
  name: "example_function",
  description: "An example function that returns a greeting",
  version: "1.0.0",
};
