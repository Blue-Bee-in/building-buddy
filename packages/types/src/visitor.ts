export type VisitorPassType = 'single_use' | 'recurring';
export type EntryType = 'pre_approved' | 'walk_in' | 'delivery' | 'domestic_help';
export type DeliveryStatus = 'pending' | 'arrived' | 'held_at_gate' | 'delivered';

export interface VisitorPass {
  id: string;
  societyId: string;
  flatId: string;
  createdByResidentId: string;
  visitorName: string;
  visitorPhone: string;
  type: VisitorPassType;
  validFrom: string;
  validUntil: string;
  qrCode: string;
  otp: string;
  usedAt: string | null;
  isActive: boolean;
}

export interface VisitorEntry {
  id: string;
  societyId: string;
  flatId: string;
  visitorPassId: string | null;
  visitorName: string;
  visitorPhone: string;
  vehicleNumber: string | null;
  photoUrl: string | null;
  entryAt: string;
  exitAt: string | null;
  guardId: string;
  type: EntryType;
}

export interface DeliveryPass {
  id: string;
  societyId: string;
  flatId: string;
  createdByResidentId: string;
  courierName: string;
  expectedFrom: string;
  expectedUntil: string;
  status: DeliveryStatus;
  packagePhotoUrl: string | null;
  arrivedAt: string | null;
  deliveredAt: string | null;
}

export interface CreateVisitorPassRequest {
  visitorName: string;
  visitorPhone: string;
  type: VisitorPassType;
  validFrom: string;
  validUntil: string;
}

export interface ValidatePassRequest {
  qrCode?: string;
  otp?: string;
}
