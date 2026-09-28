export interface User {
  id: string;
  login: string;
  name: string;
  display_name?: string;
}

export interface MeResponse {
  user: User;
}
