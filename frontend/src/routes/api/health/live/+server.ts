// Liveness probe — indicates process is alive
// SvelteKit server route

export async function GET() {
  return new Response(JSON.stringify({ alive: true }), {
    status: 200,
    headers: { 'Content-Type': 'application/json' },
  });
}