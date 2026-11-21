import { API_BASE_URL } from "$app/env/private";
import { fail, redirect } from "@sveltejs/kit";
import type { Actions } from "./$types";
import { dev } from "$app/env";

export const actions = {
  default: async ({ request, cookies }) => {
    const data = await request.formData();
    const username = data.get("username")?.toString();
    const password = data.get("password")?.toString();

    if (!username || !password) {
      return fail(400, { missing: true, message: "All fields need to be filled out", username });
    }

    try {
      const res = await fetch(`${API_BASE_URL}/api/auth/login`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ username, password }),
      });

      if (!res.ok) {
        const errData = await res.json().catch(() => ({}));
        return fail(res.status, { message: errData.message || "Login failed", username });
      }

			const body = await res.json().catch(() => null);
			const accessToken: string | undefined = body?.access_token;
			if(!accessToken) return fail(502, { message: "Invalid response from API", username });

      cookies.set("session", accessToken, { path: "/", httpOnly: true, sameSite: "lax", secure: !dev, maxAge: body.expires_in});
			cookies.set("refresh", body.refresh_token, { path: "/", httpOnly: true, sameSite: "lax", secure: !dev, maxAge: 60 * 60 * 24 * 7})

    } catch (err) {
      return fail(500, { message: "Could not connect to API service", error: err });
    }

    redirect(303, "/home");
  },
} satisfies Actions;
