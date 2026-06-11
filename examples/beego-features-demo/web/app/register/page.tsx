import RegisterForm from "@/components/RegisterForm";

export const metadata = {
  title: "Register",
};

export default function RegisterPage() {
  return (
    <div className="max-w-[400px] mx-auto py-8">
      <h1 className="text-2xl font-bold mb-6">Create Account</h1>
      <RegisterForm />
      <p className="text-sm text-[var(--color-muted)] mt-4 text-center">
        Already have an account? <a href="/login">Login</a>
      </p>
    </div>
  );
}
