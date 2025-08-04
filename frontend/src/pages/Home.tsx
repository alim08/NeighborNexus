import React from 'react';
import { Link } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';
import { 
  HeartIcon, 
  MapPinIcon, 
  ClockIcon, 
  UserGroupIcon,
  ArrowRightIcon 
} from '@heroicons/react/24/outline';

const Home: React.FC = () => {
  const { user } = useAuth();

  return (
    <div className="max-w-7xl mx-auto">
      {/* Hero Section */}
      <div className="text-center py-16 px-4 sm:px-6 lg:px-8">
        <h1 className="text-4xl font-bold tracking-tight text-gray-900 sm:text-6xl">
          Connect with Your
          <span className="text-blue-600"> Neighbors</span>
        </h1>
        <p className="mt-6 text-lg leading-8 text-gray-600 max-w-2xl mx-auto">
          NeighborNexus helps you find and offer help in your community. 
          Whether you need assistance with a small task or want to volunteer your skills, 
          we make it easy to connect with nearby neighbors.
        </p>
        <div className="mt-10 flex items-center justify-center gap-x-6">
          {user ? (
            <>
              <Link
                to="/needs/create"
                className="rounded-md bg-blue-600 px-3.5 py-2.5 text-sm font-semibold text-white shadow-sm hover:bg-blue-500 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue-600"
              >
                Ask for Help
              </Link>
              <Link
                to="/volunteer/profile"
                className="text-sm font-semibold leading-6 text-gray-900"
              >
                Become a Volunteer <ArrowRightIcon className="inline h-4 w-4 ml-1" />
              </Link>
            </>
          ) : (
            <>
              <Link
                to="/register"
                className="rounded-md bg-blue-600 px-3.5 py-2.5 text-sm font-semibold text-white shadow-sm hover:bg-blue-500 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue-600"
              >
                Get Started
              </Link>
              <Link
                to="/login"
                className="text-sm font-semibold leading-6 text-gray-900"
              >
                Sign In <ArrowRightIcon className="inline h-4 w-4 ml-1" />
              </Link>
            </>
          )}
        </div>
      </div>

      {/* Features Section */}
      <div className="py-24 sm:py-32">
        <div className="mx-auto max-w-7xl px-6 lg:px-8">
          <div className="mx-auto max-w-2xl lg:text-center">
            <h2 className="text-base font-semibold leading-7 text-blue-600">
              How It Works
            </h2>
            <p className="mt-2 text-3xl font-bold tracking-tight text-gray-900 sm:text-4xl">
              Everything you need to help your community
            </p>
            <p className="mt-6 text-lg leading-8 text-gray-600">
              NeighborNexus uses AI-powered matching to connect people who need help 
              with those who can provide it, all while respecting your privacy and location.
            </p>
          </div>
          <div className="mx-auto mt-16 max-w-2xl sm:mt-20 lg:mt-24 lg:max-w-none">
            <dl className="grid max-w-xl grid-cols-1 gap-x-8 gap-y-16 lg:max-w-none lg:grid-cols-3">
              <div className="flex flex-col">
                <dt className="flex items-center gap-x-3 text-base font-semibold leading-7 text-gray-900">
                  <HeartIcon className="h-5 w-5 flex-none text-blue-600" aria-hidden="true" />
                  Ask for Help
                </dt>
                <dd className="mt-4 flex flex-auto flex-col text-base leading-7 text-gray-600">
                  <p className="flex-auto">
                    Describe what you need help with in plain English. Our AI understands 
                    your request and finds the best volunteers nearby.
                  </p>
                </dd>
              </div>
              <div className="flex flex-col">
                <dt className="flex items-center gap-x-3 text-base font-semibold leading-7 text-gray-900">
                  <MapPinIcon className="h-5 w-5 flex-none text-blue-600" aria-hidden="true" />
                  Privacy-First Location
                </dt>
                <dd className="mt-4 flex flex-auto flex-col text-base leading-7 text-gray-600">
                  <p className="flex-auto">
                    We use privacy-preserving location matching so you can help and be helped 
                    by nearby neighbors without revealing exact addresses.
                  </p>
                </dd>
              </div>
              <div className="flex flex-col">
                <dt className="flex items-center gap-x-3 text-base font-semibold leading-7 text-gray-900">
                  <UserGroupIcon className="h-5 w-5 flex-none text-blue-600" aria-hidden="true" />
                  Real-Time Connection
                </dt>
                <dd className="mt-4 flex flex-auto flex-col text-base leading-7 text-gray-600">
                  <p className="flex-auto">
                    Get instant notifications when someone wants to help or when your offer 
                    is accepted. Coordinate and complete tasks together.
                  </p>
                </dd>
              </div>
            </dl>
          </div>
        </div>
      </div>

      {/* CTA Section */}
      <div className="bg-blue-600">
        <div className="px-6 py-24 sm:px-6 sm:py-32 lg:px-8">
          <div className="mx-auto max-w-2xl text-center">
            <h2 className="text-3xl font-bold tracking-tight text-white sm:text-4xl">
              Ready to help your community?
            </h2>
            <p className="mx-auto mt-6 max-w-xl text-lg leading-8 text-blue-200">
              Join thousands of neighbors who are already helping each other every day.
            </p>
            <div className="mt-10 flex items-center justify-center gap-x-6">
              <Link
                to="/register"
                className="rounded-md bg-white px-3.5 py-2.5 text-sm font-semibold text-blue-600 shadow-sm hover:bg-blue-50 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-white"
              >
                Get Started Today
              </Link>
              <Link
                to="/login"
                className="text-sm font-semibold leading-6 text-white"
              >
                Sign In <ArrowRightIcon className="inline h-4 w-4 ml-1" />
              </Link>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default Home; 