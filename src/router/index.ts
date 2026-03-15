import { Hono } from "hono";
import { healthRouter } from "../modules/health";
import { usersRouter } from "../modules/users";

const router = new Hono();

router.route("/health", healthRouter);
router.route("/users", usersRouter);

export { router };
