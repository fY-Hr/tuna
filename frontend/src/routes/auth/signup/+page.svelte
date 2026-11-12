<script lang="ts">
  import tunaIcon from "../../../lib/assets/tuna-icon-open.svg";
  import FloatingInput from "../../../lib/components/FloatingInput.svelte";
  import { enhance } from "$app/forms";
  import { refreshAll } from "$app/navigation";
  import type { PageProps } from "./$types";
  import type { SubmitFunction } from "$app/forms";
  import Loader from "@lucide/svelte/icons/loader";
  import { goto } from "$app/navigation";

  let { form }: PageProps = $props();

  let username: string = $derived(form?.username ?? "");
  let email: string = $derived(form?.email ?? "");
  let password: string = $state("");
  let confirmPassword: string = $state("");
  let isPasswordMatch: boolean = $derived(password === confirmPassword);
  let status: "loading" | "idle" = $state("idle");

  const handleSubmit: SubmitFunction = () => {
    status = "loading";

    return async({ result, update }) =>{
      if(result.type === "redirect") {
				goto(result.location, { state: {accountCreated: true}})
      } else {
        status = "idle";
        await update();
      }
    }
  }

  $effect(() => {
    const handlePageShow = (event: PageTransitionEvent) => {
      if(event.persisted) {
        refreshAll();
      }
    };

    window.addEventListener("pageshow", handlePageShow);
    return () => window.removeEventListener("pageshow", handlePageShow);
  });
</script>

<div class="w-screen h-dvh flex justify-center items-center">
  <div class="w-full h-full flex flex-col justify-end items-center md:h-auto md:w-2xl md:flex-row md:items-stretch md:bg-white md:rounded-xl md:p-2 md:pb-0 text-gray-700 md:shadow-xl">

    <div class="w-full flex flex-col rounded-t-[3rem] md:rounded-none md:w-1/2 md:p-8 bg-white h-4/6 md:h-auto">
      <div class="flex flex-col items-center pt-7 md:pt-0 h-full">
        <div class="flex flex-col items-center md:hidden text-gray-700">
          <h1 class="text-3xl font-bold">Welcome to <span class="text-[#7BA0FF]">tuna</span>!</h1>
          <p class="text-md">Dive into various wave.</p>
        </div>

          <form method="POST" use:enhance={handleSubmit} class="w-11/12 md:w-full flex items-center p-8 md:p-0 flex-col gap-4">
            {#if form?.message}
              <p class="text-red-500 text-sm">{form.message}</p>
            {/if}
            <div class="flex flex-col gap-4 w-full">
              <FloatingInput id="username" name="username" label="Username" bind:value={username} invalid={form?.missing && !username} />
              <FloatingInput id="email" name="email" label="Email" bind:value={email} invalid={form?.missing && !email} />
              <FloatingInput id="password" name="password" type="password" label="Password" bind:value={password} invalid={form?.missing && !password} />
              <FloatingInput id="confirmPassword" name="confirmPassword" type="password" label="Confirm Password" bind:value={confirmPassword} invalid={!isPasswordMatch && !!password} oninput={() => isPasswordMatch} />
            </div>

            <div class="w-full">
              <button
                class="w-full md:py-2 py-3 font-semibold bg-[#7BA0FF] text-white rounded-full transition-all duration-100 hover:cursor-pointer hover:bg-white hover:text-[#7BA0FF] border-2 border-[#7BA0FF] flex justify-center"
								disabled={status === "loading"}
              >
							{#if status === "loading"}
								<div class="flex h-full items-center text-sm">
									<Loader class="animate-spin" size={18}/>
								</div>
							{:else}
								Sign Up
							{/if}
              </button>
            </div>

            <div>
              <p class="text-sm text-gray-400">Already have an account? <a href="/auth/login" class="text-[#7BA0FF] underline">Log in</a></p>
            </div>
          </form>
      </div>
    </div>

    <div class="hidden md:flex w-1/2 p-8 pb-0 flex-col justify-between">
      <div class="flex flex-col">
        <h1 class="text-4xl font-bold">Welcome to <span class="text-[#7BA0FF]">tuna</span>!</h1>
        <p class="text-md">Dive into various wave.</p>
      </div>
      <div class="flex justify-center relative">
        <enhanced:img src={tunaIcon} alt="Tuna Icon" class="w-40" />
      </div>
    </div>

  </div>
</div>
