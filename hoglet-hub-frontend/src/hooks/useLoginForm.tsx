import { useRouter } from "next/navigation";
import { useState } from "react";
import { useAuth } from "../contexts/AuthContext";

export interface LoginFormData {
  email: string;
  password: string;
}

export type AuthError = Error & {
  status?: number;
};

export function useLoginForm() {
  const { login } = useAuth();
  const router = useRouter();
  const [error, setError] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(false);

  const handleLogin = async (): Promise<void> => {
    setIsLoading(true);
    setError(null);

    try {
      // Simulate API call delay
      await new Promise((resolve) => setTimeout(resolve, 800));

      // TODO: Replace with actual authentication call to your backend
      const mockToken =
        "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c";

      login(mockToken);
      router.push("/");
    } catch (err) {
      const authError = err as AuthError;
      setError(authError.message || "Authentication failed");
    } finally {
      setIsLoading(false);
    }
  };

  return { handleLogin, error, isLoading };
}
