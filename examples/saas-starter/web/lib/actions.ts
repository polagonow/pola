"use server";

import { AuthAction, TeamAction, PaymentsAction } from "@pola/actions";

export async function signIn(email: string, password: string) {
  return AuthAction.signIn(email, password);
}

export async function signUp(email: string, password: string, inviteId: string) {
  return AuthAction.signUp(email, password, inviteId);
}

export async function signOut() {
  return AuthAction.signOut();
}

export async function updatePassword(current: string, next: string, confirm: string) {
  return AuthAction.updatePassword(current, next, confirm);
}

export async function updateAccount(name: string, email: string) {
  return AuthAction.updateAccount(name, email);
}

export async function deleteAccount(password: string) {
  return AuthAction.deleteAccount(password);
}

export async function inviteTeamMember(email: string, role: string) {
  return TeamAction.inviteTeamMember(email, role);
}

export async function removeTeamMember(memberId: number) {
  return TeamAction.removeTeamMember(memberId);
}

export async function checkout(priceId: string) {
  return PaymentsAction.checkout(priceId);
}

export async function customerPortal() {
  return PaymentsAction.customerPortal();
}
