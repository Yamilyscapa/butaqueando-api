import { Hono } from "hono";
import { getHealth } from "./health.service";

const healthRouter = new Hono();

healthRouter.get("/", (c) => c.json(getHealth()));

export { healthRouter };
