export type ComplaintCategory =
  | 'plumbing'
  | 'electrical'
  | 'lift'
  | 'cleanliness'
  | 'security'
  | 'parking'
  | 'internet'
  | 'other';

export type ComplaintStatus = 'open' | 'in_progress' | 'resolved' | 'closed';

export interface Complaint {
  id: string;
  societyId: string;
  flatId: string;
  raisedByResidentId: string;
  category: ComplaintCategory;
  description: string;
  photoUrls: string[];
  status: ComplaintStatus;
  assignedTo: string | null;
  expectedResolutionDate: string | null;
  slaBreachedAt: string | null;
  createdAt: string;
  updatedAt: string;
  resolvedAt: string | null;
  closedAt: string | null;
}

export interface CreateComplaintRequest {
  category: ComplaintCategory;
  description: string;
  photoUrls?: string[];
}

export interface AssignComplaintRequest {
  assignedTo: string;
  expectedResolutionDate: string;
}

export interface Notice {
  id: string;
  societyId: string;
  postedByUserId: string;
  title: string;
  body: string;
  category: 'maintenance' | 'event' | 'rules' | 'urgent';
  expiresAt: string | null;
  createdAt: string;
}
