export type UserRole =
  | 'super_admin'
  | 'society_admin'
  | 'committee_member'
  | 'security_guard'
  | 'resident';

export interface AuthUser {
  id: string;
  societyId: string;
  flatId: string | null;
  role: UserRole;
  name: string;
  phone: string;
  email: string | null;
}

export interface LoginRequest {
  phone: string;
  password: string;
}

export interface LoginResponse {
  accessToken: string;
  refreshToken: string;
  user: AuthUser;
}

export interface RefreshRequest {
  refreshToken: string;
}

export interface RefreshResponse {
  accessToken: string;
}
