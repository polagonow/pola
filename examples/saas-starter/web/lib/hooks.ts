"use client";

import useSWR from "swr";

export type User = {
  id: number;
  name: string;
  email: string;
  role: string;
  createdAt?: string;
  updatedAt?: string;
};

export type TeamMember = {
  id: number;
  userId: number;
  role: string;
  name: string;
  email: string;
};

export type Team = {
  id: number;
  name: string;
  stripeCustomerId?: string | null;
  stripeSubscriptionId?: string | null;
  stripeProductId?: string | null;
  planName?: string | null;
  subscriptionStatus?: string | null;
  members: TeamMember[];
};

export type ActivityLog = {
  id: number;
  teamId: number;
  userId?: number;
  action: string;
  timestamp: string;
  ipAddress?: string;
};

export const fetcher = (url: string) => fetch(url).then((res) => res.json());

export function useUser() {
  return useSWR<User | null>("/api/user", fetcher);
}

export function useTeam() {
  return useSWR<Team | null>("/api/team", fetcher);
}

export function useActivity() {
  return useSWR<ActivityLog[]>("/api/activity", fetcher);
}
