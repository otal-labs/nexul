import { Navigate, useParams } from "react-router";

// /services/:id is the pre-rename path (spec §10); every caller lands on the stack page instead. The two ids
// are the same value — a stack's id didn't change, only the route and the model backing it did.
export const ServicePage = () => {
  const { serviceId } = useParams();
  return <Navigate to={`/stacks/${serviceId}`} replace />;
};
