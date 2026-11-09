<script lang="ts">
  import tunaIcon from "../../../lib/assets/tuna-icon.svg";
  import FloatingInput from "../../../lib/components/FloatingInput.svelte";
  import { enhance } from "$app/forms";
  import { refreshAll, goto } from "$app/navigation";
  import type { PageProps } from "./$types";
	import type { SubmitFunction } from "$sveltejs/kit";
  import { page } from "$app/state";
  import { fly, fade } from "svelte/transition";
  import CircleCheck from "@lucide/svelte/icons/circle-check";
	import Loader from "@lucide/svelte/icons/loader";

  let { form }: PageProps = $props();

  let username: string = $derived(form?.username ?? "");
  let password: string = $state("");
	let status: "idle" | "loading" = $state("idle");

  let isAccountCreated: boolean = $state(false);

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
    if (page.state.accountCreated) {
      isAccountCreated = true;
      const timeout = setTimeout(() => {
        isAccountCreated = false;
      },4000);
      page.state.accountCreated = false;
      return () => clearTimeout(timeout);
    }
  });

  $effect(() => {
    const handlePageshow = (event: PageTransitionEvent) => {
      if (event.persisted) {
        refreshAll();
      }
    };

    window.addEventListener("pageshow", handlePageshow);
    return () => window.removeEventListener("pageshow", handlePageshow);
  });
</script>

<div class="w-screen h-dvh flex justify-center items-center flex-col">
  {#if isAccountCreated}
    <div class="fixed inset-x-0 top-6 z-50 flex justify-center pointer-events-none">
      <div
        in:fly={{ y: 20, duration: 300 }}
        out:fade={{ duration: 300 }}
        class="p-4 bg-white text-[#7BA0FF] rounded-full flex items-center justify-center gap-2"
      >
        <CircleCheck color="#7BA0FF" size={18} /> Account created successfully!
      </div>
    </div>
  {/if}
  <div class="w-full h-full flex flex-col justify-end items-center md:h-auto md:w-2xl md:flex-row md:bg-white md:rounded-xl md:p-2 md:pb-0 text-gray-700 md:shadow-xl">
    <div class="w-full h-4/6 bg-white flex flex-col rounded-t-[3rem] md:rounded-none md:w-1/2 md:h-full md:p-8">
      <div class="flex flex-col items-center pt-7 md:pt-0">
        <div class="flex flex-col items-center md:hidden">
          <h1 class="text-3xl font-bold">Let's dive to the <span class="text-[#7BA0FF]">wave</span>!</h1>
          <p class="text-md">Dive into various waves.</p>
        </div>

        <form method="POST" use:enhance={handleSubmit} class="w-11/12 md:w-full flex items-center p-8 md:p-0 flex-col gap-4">
          {#if form?.message}
            <p class="text-red-500 text-sm">{form.message}</p>
          {/if}
          <div class="flex flex-col gap-4 w-full">
            <FloatingInput id="username" name="username" label="Username" bind:value={username} invalid={form?.missing && !username} />
            <FloatingInput id="password" name="password" type="password" label="Password" bind:value={password} invalid={form?.missing && !password} />
          </div>

          <div class="w-full">
            <button
              class="w-full md:py-2 py-3 font-semibold bg-[#7BA0FF] text-white rounded-full transition-all duration-100 hover:cursor-pointer hover:bg-white hover:text-[#7BA0FF] border-2 border-[#7BA0FF] flex justify-center"
							disabled={status === "loading"}
            >
						{#if status == "loading"}
							<div class="flex h-full items-center text-sm">
								<Loader class="animate-spin" size={18}/>
							</div>
						{:else}
							Log in
						{/if}
            </button>
          </div>

          <div>
            <p class="text-sm text-gray-400">New here? <a href="/auth/signup" class="text-[#7BA0FF] underline">Create an account</a></p>
          </div>
        </form>
      </div>
    </div>

    <div class="hidden md:flex w-1/2 h-full p-8 pb-0 flex-col justify-between gap-3">
      <div class="flex flex-col text-gray-700">
        <h1 class="text-4xl font-bold">Let's dive to the <span class="text-[#7BA0FF]">wave</span>!</h1>
        <p class="text-md">Dive into various waves.</p>
      </div>
      <div class="flex justify-center relative">
        <enhanced:img src={tunaIcon} alt="Tuna Icon" class="w-40" />
      </div>
    </div>

  </div>
</div>
