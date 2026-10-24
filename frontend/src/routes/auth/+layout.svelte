<script lang="ts">
  import type { LayoutProps } from './$types';
  import tunaIconOpen from "../../lib/assets/tuna-icon-open.svg"
  let { children }: LayoutProps = $props();
</script>

<div class="bg-[#7BA0FF] w-screen h-screen overflow-hidden">
  <div class="absolute top-24 md:top-auto left-12 mr-12 z-10 md:bottom-0 md:left-10 md:mr-10">
    <enhanced:img src={tunaIconOpen} alt="" class="w-50 md:w-md md:h-md"/>
  </div>
  <div class="z-20 absolute">
    {@render children()}
  </div>
</div>
