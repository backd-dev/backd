export default async function handler(req: Request): Promise<Response> {
  return new Response(
    JSON.stringify({ message: "hello" }),
    { headers: { "Content-Type": "application/json" } },
  );
}
