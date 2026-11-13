import { redirect } from "@sveltejs/kit";
import type { PageServerLoad } from "./$types";

export const load: PageServerLoad = async ({ cookies }) => {
  const token = cookies.get("session");
  if (!token) {
    redirect(303, "/auth/login");
  } else {
    redirect(303, "/home");
  }
};
