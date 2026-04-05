export type ResidentRole = 'owner' | 'tenant';
export type VehicleType = '2-wheeler' | '4-wheeler';
export type DomesticHelpType = 'maid' | 'cook' | 'driver' | 'nanny' | 'other';

export interface Flat {
  id: string;
  societyId: string;
  block: string | null;
  floor: number;
  unit: string;
  type: string;
}

export interface Resident {
  id: string;
  flatId: string;
  societyId: string;
  name: string;
  phone: string;
  email: string | null;
  role: ResidentRole;
  moveInDate: string;
  moveOutDate: string | null;
  isActive: boolean;
}

export interface Vehicle {
  id: string;
  residentId: string;
  flatId: string;
  type: VehicleType;
  numberPlate: string;
  make: string | null;
  color: string | null;
}

export interface DomesticHelp {
  id: string;
  flatId: string;
  societyId: string;
  name: string;
  phone: string;
  photoUrl: string | null;
  type: DomesticHelpType;
  workingDays: number[]; // 0=Sunday, 1=Monday, ...
  entryTime: string;     // "HH:MM"
  exitTime: string;      // "HH:MM"
  isActive: boolean;
  isBlacklisted: boolean;
}

export interface InviteResidentRequest {
  flatId: string;
  name: string;
  phone: string;
  email?: string;
  role: ResidentRole;
  moveInDate: string;
}
