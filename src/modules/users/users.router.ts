import { Hono } from "hono";
import { createUser, getUsers } from "./users.service";
import { HttpError } from "../../shared/errors/http-error";

const usersRouter = new Hono();

usersRouter.get("/", (c) => c.json(getUsers()));

usersRouter.post("/", async (c) => {
  const body = await c.req.json();
  const name = body?.name;
  const email = body?.email;

  if (!name || !email) {
    throw new HttpError(400, "name and email are required");
  }

  const user = createUser({ name, email });
  return c.json(user, 201);
});

export { usersRouter };
