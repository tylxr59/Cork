export interface User {
  id: string;
  username: string;
  is_admin: boolean;
  created_at: string;
}

export interface Cover {
  url: string;
  color: string;
}

export interface Board {
  id: string;
  name: string;
  description: string;
  is_public: boolean;
  pin_count: number;
  covers: Cover[];
  owner: string;
  can_edit: boolean;
  created_at: string;
  updated_at: string;
}

export interface Pin {
  id: string;
  board_id: string;
  board_name: string;
  owner: string;
  title: string;
  description: string;
  source_url: string;
  /** Price when the pin was saved, as a plain decimal string ("" if none). */
  price: string;
  /** ISO 4217 code ("" if unknown). */
  currency: string;
  image_url: string;
  thumb_url: string;
  width: number;
  height: number;
  color: string;
  created_at: string;
  updated_at: string;
  can_edit: boolean;
}

export interface Status {
  needs_setup: boolean;
  registration_open: boolean;
  user: User | null;
}

export interface ScrapedPage {
  url: string;
  title: string;
  description: string;
  site_name: string;
  price: string;
  currency: string;
  images: string[];
}

export interface AdminUser extends User {
  board_count: number;
  pin_count: number;
}
