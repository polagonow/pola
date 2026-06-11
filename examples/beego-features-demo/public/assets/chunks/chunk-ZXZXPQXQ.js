// utils/csrf.ts
function csrfToken() {
  return document.querySelector('meta[name="csrf-token"]')?.content ?? "";
}

export {
  csrfToken
};
