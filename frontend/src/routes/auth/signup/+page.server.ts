import { API_BASE_URL } from "$app/env/private";
import { fail, redirect } from "@sveltejs/kit";
import type { Actions, PageServerLoad } from "./$types";

export const load: PageServerLoad = async ({ cookies }) => {
  const token = cookies.get("session");
  if (token) {
    return redirect(303, "/home");
  }
};

export const actions = {
  default: async ({ request }) => {
    const data = await request.formData();
    const username = data.get("username")?.toString();
    const email = data.get("email")?.toString();
    const password = data.get("password")?.toString();
    const confirmPassword = data.get("confirmPassword")?.toString();

    if (!username || !email || !password) {
      return fail(400, {
        missing: true,
        message: "All fields need to be filled out",
        username,
        email,
      });
    }

    if (password !== confirmPassword) {
      return fail(400, { message: "Passwords do not match", username, email });
    }

    try {
      const res = await fetch(`${API_BASE_URL}/api/auth/register`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ username, email, password }),
      });

      if (!res.ok) {
        const errData = await res.json().catch(() => ({}));
        return fail(res.status, {
          message: errData.message || "Registration failed",
          username,
          email,
        });
      }
    } catch (err) {
      return fail(500, { message: "Registration failed", error: err, username, email});
    }

    redirect(303, "/auth/login");
  },
} satisfies Actions;
