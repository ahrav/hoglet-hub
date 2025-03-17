# Hoglet Hub Frontend

A Next.js frontend for the Hoglet Hub tenant provisioning system. This application allows you to create tenants, monitor operations, and manage provisioning tasks.

## Features

- **Tenant Creation**: Create new tenants with customized settings
- **Operations Monitoring**: Track the status of ongoing operations
- **Authentication**: Secure access to your provisioning system
- **Real-time Updates**: Follow the progress of tenant provisioning tasks

## Prerequisites

- Node.js 18.x or later
- npm or yarn
- Access to the Hoglet Hub API
- Docker (for Kubernetes deployment)
- kubectl and kind (for local Kubernetes development)

## Getting Started

### Local Development (Outside Kubernetes)

1. Clone the repository
2. Navigate to the project directory:
   ```
   cd hoglet-hub-frontend
   ```
3. Install dependencies:
   ```
   npm install
   ```
4. Configure environment variables:
   Create or modify the `.env.local` file in the project root with the following variables:
   ```
   NEXT_PUBLIC_API_URL=http://api.hoglet-hub.local
   NEXT_PUBLIC_DEV_MODE=true
   PORT=4000
   ```
5. Generate API client:
   ```
   npm run generate-api
   ```
6. Start the development server:
   ```
   npm run dev
   ```
   The application will be available at [http://localhost:4000](http://localhost:4000).

### Kubernetes Deployment

This application is designed to run within a Kubernetes cluster. The project includes all necessary Kubernetes manifests in the `k8s/dev/frontend` directory.

To deploy to the local kind cluster:

1. Build the Docker image:
   ```
   cd hoglet-hub
   make docker-frontend
   ```

2. Load the image into the kind cluster:
   ```
   make dev-load
   ```

3. Apply the Kubernetes manifests:
   ```
   make dev-apply
   ```

4. Access the frontend via the ingress:

   Update your hosts file to include:
   ```
   127.0.0.1 hoglet-hub.local
   ```

   Then access the application at [http://hoglet-hub.local](http://hoglet-hub.local)

5. For development, you can port-forward the service:
   ```
   make frontend-port-forward
   ```
   Then access at [http://localhost:4000](http://localhost:4000)

## Development Workflow

For a complete development workflow:

1. Run `make dev-all` to set up the entire development environment
2. Make changes to the frontend code
3. Run `make docker-frontend` to build the new Docker image
4. Run `make dev-load` to load the image into the kind cluster
5. Run `make rollout-restart-frontend` to apply the changes

## API Integration

The frontend is designed to work with the Hoglet Hub API. It expects the following endpoints:

- `POST /api/v1/tenants` - Create a new tenant
- `DELETE /api/v1/tenants/{tenant_id}` - Delete a tenant
- `GET /api/v1/operations/{operation_id}` - Get operation details

## Technologies Used

- **Next.js** - React framework with server-side rendering
- **TypeScript** - Type-safe JavaScript
- **TailwindCSS** - Utility-first CSS framework
- **React Hook Form** - Form validation
- **Zod** - Schema validation
- **React Query** - Data fetching
- **Axios** - HTTP client

## Learn More

To learn more about Next.js, take a look at the following resources:

- [Next.js Documentation](https://nextjs.org/docs) - learn about Next.js features and API.
- [Learn Next.js](https://nextjs.org/learn) - an interactive Next.js tutorial.

You can check out [the Next.js GitHub repository](https://github.com/vercel/next.js) - your feedback and contributions are welcome!

## Deploy on Vercel

The easiest way to deploy your Next.js app is to use the [Vercel Platform](https://vercel.com/new?utm_medium=default-template&filter=next.js&utm_source=create-next-app&utm_campaign=create-next-app-readme) from the creators of Next.js.

Check out our [Next.js deployment documentation](https://nextjs.org/docs/app/building-your-application/deploying) for more details.
