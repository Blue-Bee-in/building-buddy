export type SocietyStatus = 'draft' | 'kyc_pending' | 'live';
export type ChargeFrequency = 'monthly' | 'quarterly' | 'annual';

export interface Society {
  id: string;
  name: string;
  address: string;
  city: string;
  state: string;
  pincode: string;
  razorpayAccountId: string | null;
  status: SocietyStatus;
  createdAt: string;
  updatedAt: string;
}

export interface ChargeHead {
  id: string;
  societyId: string;
  name: string;
  amount: number;
  frequency: ChargeFrequency;
  applicableFlatTypes: string[];
  isActive: boolean;
}

export interface CreateSocietyRequest {
  name: string;
  address: string;
  city: string;
  state: string;
  pincode: string;
}

export interface BankDetails {
  accountHolderName: string;
  accountNumber: string;
  ifscCode: string;
  bankName: string;
}
