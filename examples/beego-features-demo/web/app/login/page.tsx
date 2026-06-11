import LoginForm from "@/components/LoginForm";

export const metadata = {
  title: "Login",
};

export default function LoginPage() {
  return (
    <div className="max-w-[400px] mx-auto py-8">
      <h1 className="text-2xl font-bold mb-6">Login</h1>
      <LoginForm />
      <p className="text-sm text-[var(--color-muted)] mt-4 text-center">
        Don't have an account? <a href="/register">Register</a>
      </p>
    </div>
  );
}
