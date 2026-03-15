import {
  createUser as createUserRecord,
  listUsers as listUserRecords,
  type User,
} from "./users.repository";

type CreateUserInput = {
  name: string;
  email: string;
};

export function getUsers(): User[] {
  return listUserRecords();
}

export function createUser(input: CreateUserInput): User {
  return createUserRecord(input);
}
