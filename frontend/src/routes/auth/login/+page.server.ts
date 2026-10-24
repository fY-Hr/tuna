import { API_BASE_URL } from "$app/env/private";
import { fail, redirect } from "@sveltejs/kit";
import type { Actions } from "./$types";

export const actions = {
  default: async ({ request, cookies }) => {
    const data = await request.formData();
    const username = data.get("username")?.toString();
    const password = data.get("password")?.toString();

    if (!username || !password) {
      return fail(400, { missing: true, message: "Username and password required", username });
    }

    try {
      const res = await fetch(`${API_BASE_URL}/api/auth/login`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ username, password })
      });

      if (!res.ok) {
        const errData = await res.json().catch(() => ({}));
        return fail(res.status, { message: errData.message || "Login failed", username });
      }

      const { token } = await res.json();

      cookies.set("session", token, {
        path: "/",
        httpOnly: true,
        sameSite: "lax",
        secure: true,
        maxAge: 60 * 60 * 24 * 7
      });
    } catch (err) {
      return fail(500, { message: "Could not connect to API service" })
    }

    redirect(303, "/home")
  }
} satisfies Actions;
