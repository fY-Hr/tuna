import { redirect } from "@sveltejs/kit";
import type { LayoutServerLoad } from "./$types";

// taking the isAuthenticated from the parent (the routes/+layout-server.ts) bcs it return the check result
export const load: LayoutServerLoad = async ({ parent }) => {
  const { isAuthenticated }= await parent();
  if (isAuthenticated) {
		redirect(303, "/home");
	}
};
