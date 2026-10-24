import { redirect } from "@sveltejs/kit";

export async function load({ cookies }) {
  const token = cookies.get("session");
  if (!token) {
    redirect(302, "/auth/login")
  } else {
    redirect(302, "/home")
  }
}
