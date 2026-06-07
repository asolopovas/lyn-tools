export function errorMessage(error: unknown, fallback: string): string {
  return error instanceof Error ? error.message : fallback;
}

export function errorDetail(error: unknown): string {
  return errorMessage(error, String(error));
}
