/**
 * Home Page
 * 
 * Landing page for your app. Customize as needed.
 */
import { useAuth } from '@digistratum/ds-core';
import config from '../config';

export function HomePage() {
  const { user, isLoading } = useAuth();

  if (isLoading) {
    return (
      <div className="flex items-center justify-center min-h-[50vh]">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
      </div>
    );
  }

  return (
    <div className="container mx-auto px-4 py-8">
      <h1 className="text-3xl font-bold mb-4">
        Welcome to {config.name}
      </h1>
      
      {user ? (
        <div className="space-y-4">
          <p className="text-lg">
            Hello, <span className="font-semibold">{user.name || user.email}</span>!
          </p>
          <p className="text-gray-600 dark:text-gray-400">
            You're signed in and ready to go.
          </p>
        </div>
      ) : (
        <div className="space-y-4">
          <p className="text-gray-600 dark:text-gray-400">
            Please sign in to access all features.
          </p>
        </div>
      )}
    </div>
  );
}

export default HomePage;
