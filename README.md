# More information about the authorization module

The authorization module provides an API for:

- Registering new users;
- Deleting users;
- Changing the roles assigned to a user;
- Changing the user's data;
- Checking if a user exists;
- Checking a user's roles.

 Only application modules should have access to the API. It is important to consider the possibility of adding more modules in the future.

## Registering new users

A user is considered to be registered if the following information is known about them on the server:

- GitHub `id`;
- Telegram `id`;
- List of roles;
- User data.

 User data includes:

- Last name, first name, and middle name;
- Full group number.

 User data may not be set during registration and may be changed later.

User data must be stored in a database of any type (except those that store data exclusively in RAM).

### Roles

There are 3 types of roles:

- Student. The role is assigned to an
