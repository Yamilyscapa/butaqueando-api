import { Hono } from "hono";
import { router } from "./router";
import { errorHandler } from "./shared/middleware/error-handler";

const app = new Hono();

app.onError(errorHandler);

app.get("/", (c) => c.json({ name: "hono-bun-api", status: "ok" }));
app.route("/", router);

export { app };
