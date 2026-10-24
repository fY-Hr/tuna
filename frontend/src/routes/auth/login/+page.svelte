<script lang="ts">
  import tunaIcon from "../../../lib/assets/tuna-icon.svg";
  import FloatingInput from "../../../lib/components/FloatingInput.svelte";
  import { enhance } from "$app/forms";
  import type { PageProps } from "./$types";

  let { form }: PageProps = $props();

  let username: string = $derived(form?.username ?? "");
  let password: string = $state("");
</script>

<div class="w-screen h-screen flex justify-center items-center">
  <div class="w-full h-full flex flex-col justify-end items-center md:h-auto md:w-2xl md:flex-row md:bg-white md:rounded-xl md:p-2 md:pb-0 md:text-gray-700 md:shadow-xl">

    <div class="w-full h-4/6 bg-white flex flex-col rounded-t-[3rem] md:rounded-none md:w-1/2 md:h-full md:p-8">
      <div class="flex flex-col items-center pt-7 md:pt-0">
        <div class="flex flex-col items-center md:hidden">
          <h1 class="text-3xl font-bold">Welcome to <span class="text-[#7BA0FF]">tuna</span>!</h1>
          <p class="text-md">Dive into various wave.</p>
        </div>

        <form method="POST" use:enhance class="w-11/12 md:w-full flex items-center p-8 md:p-0 flex-col gap-4">
          {#if form?.message}
            <p class="text-red-500 text-sm">{form.message}</p>
          {/if}
          <div class="flex flex-col gap-4 w-full">
            <FloatingInput id="username" name="username" label="Username" bind:value={username} invalid={form?.missing && !username} />
            <FloatingInput id="password" name="password" type="password" label="Password" bind:value={password} invalid={form?.missing && !password} />
          </div>

          <div class="w-full">
            <button
              class="w-full md:py-2 py-3 font-semibold bg-[#7BA0FF] text-white rounded-full transition-all duration-100 hover:cursor-pointer hover:bg-white hover:text-[#7BA0FF] border-2 border-[#7BA0FF]"
            >
              Log in
            </button>
          </div>

          <div>
            <p class="text-sm text-gray-400">New here? <a href="/auth/register" class="text-[#7BA0FF] underline">Register</a></p>
          </div>
        </form>
      </div>
    </div>

    <div class="hidden md:flex w-1/2 h-full p-8 pb-0 flex-col justify-between gap-3">
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
