import { redirect } from "@sveltejs/kit";
import type { PageServerLoad } from "./$types";

export const load: PageServerLoad = async ({ parent }) => {
	const { isAuthenticated } = await parent();
	redirect(303, isAuthenticated ? '/home' : '/auth/login');
}
