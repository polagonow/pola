"use server";

import di from "@pola/di";

export async function submitContact(name: string, email: string, message: string) {
  const result = await di.submitContact(name, email, message);
  return result;
}
