type HealthResponse = {
  ok: boolean;
  timestamp: string;
};

export function getHealth(): HealthResponse {
  return {
    ok: true,
    timestamp: new Date().toISOString(),
  };
}
