// Client-side error tracking with correlation IDs
// Format: cp-<uuid>-<timestamp>

export function generateCorrelationId(): string {
  const uuid = crypto.randomUUID();
  const timestamp = Date.now();
  return `cp-${uuid}-${timestamp}`;
}

export interface ErrorContext {
  componentStack?: string;
  userAgent?: string;
  location?: string;
}

export function trackError(error: unknown, context: ErrorContext = {}) {
  const correlationId = generateCorrelationId();
  const userAgent = context.userAgent ?? (typeof navigator !== 'undefined' ? navigator.userAgent : 'unknown');
  const location = context.location ?? (typeof window !== 'undefined' ? window.location.href : 'unknown');

  console.error(`[${correlationId}] Uncaught error:`, error, {
    userAgent,
    location,
    componentStack: context.componentStack,
  });

  return correlationId;
}