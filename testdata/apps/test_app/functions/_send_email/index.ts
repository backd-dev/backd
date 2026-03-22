export default async function handler(req: Request): Promise<Response> {
  const body = await req.json();
  // Simulate email sending — private function for job tests
  return new Response(
    JSON.stringify({ sent: true, to: body.to || "test@example.com" }),
    { headers: { "Content-Type": "application/json" } },
  );
}
