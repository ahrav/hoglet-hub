'use client';

import TenantCreateForm from '../../../components/TenantCreateForm';
import { useAuth } from '../../../contexts/AuthContext';
import { useRouter } from 'next/navigation';
import { useEffect } from 'react';

export default function CreateTenantPage() {
  const { isAuthenticated } = useAuth();
  const router = useRouter();

  // Redirect to login if not authenticated
  useEffect(() => {
    if (!isAuthenticated) {
      router.push('/login');
    }
  }, [isAuthenticated, router]);

  return (
    <div className="flex flex-col items-center">
      <div className="w-full max-w-3xl">
        <h1 className="text-3xl font-bold mb-8 text-center">Create New Tenant</h1>
        <TenantCreateForm />
      </div>
    </div>
  );
}
