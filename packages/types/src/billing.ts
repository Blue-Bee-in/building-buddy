export type InvoiceStatus = 'pending' | 'partially_paid' | 'paid' | 'overdue';
export type PaymentMode = 'upi' | 'cash' | 'cheque' | 'bank_transfer';
export type PaymentStatus = 'pending' | 'success' | 'failed';
export type LineItemType = 'charge' | 'penalty';

export interface InvoiceLineItem {
  chargeHeadId: string;
  label: string;
  amount: number;
  type: LineItemType;
}

export interface Invoice {
  id: string;
  societyId: string;
  flatId: string;
  residentId: string;
  billingPeriod: string; // "YYYY-MM"
  dueDate: string;
  status: InvoiceStatus;
  totalAmount: number;
  paidAmount: number;
  pdfUrl: string | null;
  lineItems: InvoiceLineItem[];
  createdAt: string;
  updatedAt: string;
}

export interface Payment {
  id: string;
  invoiceId: string;
  amount: number;
  mode: PaymentMode;
  status: PaymentStatus;
  reference: string | null;
  notes: string | null;
  recordedBy: string | null; // admin user id for manual entries
  createdAt: string;
}

export interface RecordManualPaymentRequest {
  invoiceId: string;
  amount: number;
  mode: 'cash' | 'cheque';
  reference?: string;
  notes?: string;
  paymentDate: string;
}
