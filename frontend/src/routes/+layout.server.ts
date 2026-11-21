import type { LayoutServerLoad } from "./$types";

export const load: LayoutServerLoad = async ({ cookies }) => {
  const isAuthenticated = cookies.get("session");
	return { isAuthenticated };
};
