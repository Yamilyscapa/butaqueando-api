import type { Context } from "hono";
import { HttpError } from "../errors/http-error";

export function errorHandler(err: Error, c: Context): Response {
  if (err instanceof HttpError) {
    return c.json({ error: err.message }, err.status);
  }

  console.error(err);
  return c.json({ error: "Internal Server Error" }, 500);
}
