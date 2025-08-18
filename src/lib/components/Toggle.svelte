<script lang="ts">
  export let id: string = '';
  export let checked: boolean = false;
  export let disabled: boolean = false;
  // Optional event handlers to plug in custom behaviors (e.g., elevation flow)
  export let onClick: ((e: MouseEvent) => void) | undefined = undefined;
  export let onChange: ((e: Event) => void) | undefined = undefined;
  // Dim the track when disabled (used by Admin toggle when AutoStart is off)
  export let dimWhenDisabled: boolean = false;
</script>

<label class="relative inline-flex items-center cursor-pointer select-none">
  <input
    type="checkbox"
    id={id}
    class="sr-only"
    {disabled}
    on:click={(e: MouseEvent) => {
      if (!onClick) return;
      e.preventDefault();
      e.stopPropagation();
      onClick(e);
    }}
    on:change={(e) => onChange ? onChange(e) : null}
    bind:checked={checked}
  />
  <span
    class="block w-10 h-6 rounded-full transition-colors duration-200 relative"
    class:bg-zinc-200={!checked}
    class:bg-amber-600={checked}
    class:dark:bg-neutral-800={!checked}
    class:dark:bg-amber-500={checked}
    class:opacity-50={dimWhenDisabled}
  >
    <span
      class="block w-4 h-4 mt-1 ml-1 rounded-full transition-transform duration-200 absolute"
      class:bg-white={!checked}
      class:dark:bg-white={!checked}
      class:bg-zinc-100={checked}
      class:dark:bg-neutral-800={checked}
      style={`transform: translateX(${checked ? '1.0rem' : '0'});`}
    ></span>
  </span>
</label>
