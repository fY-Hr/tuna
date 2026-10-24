<script lang="ts">
  import type { HTMLInputAttributes } from 'svelte/elements';

  interface Props extends HTMLInputAttributes {
    id: string;
    label: string;
    name?: string;
    value?: string;
    invalid?: boolean;
  }

  let { id, label, name = id, value = $bindable(''), invalid = false, type='text', class: className = '', ...rest }: Props = $props();
</script>

<div class="relative {className}">
  <input
    {id}
    {type}
    {name}
    placeholder=" "
    aria-invalid={invalid}
    class="peer flex py-3 md:py-2 px-2 text-md md:text-sm rounded-md border-2  transition-all duration-200 outline-0 w-full
    {invalid? 'border-red-500 focus:border-[#7BA0FF]' : 'border-gray-400 focus:border-[#7BA0FF]'}"
    bind:value
    {...rest}
  />
  <label
    for={id}
    class="pointer-events-none absolute left-2 top-1/2 -translate-y-1/2 bg-white px-1 text-md md:text-sm text-gray-400 transition-all duration-200 peer-focus:top-0 peer-focus:text-xs peer-focus:text-[#7BA0FF] peer-not-placeholder-shown:top-0 peer-not-placeholder-shown:text-xs"
  >
    {label}
  </label>
</div>
