export type User = {
  id: string;
  name: string;
  email: string;
};

const users: User[] = [];

export function listUsers(): User[] {
  return users;
}

export function createUser(data: Omit<User, "id">): User {
  const user: User = {
    id: crypto.randomUUID(),
    ...data,
  };

  users.push(user);
  return user;
}
