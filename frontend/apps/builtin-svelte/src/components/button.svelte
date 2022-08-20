<script lang="ts">
  export let type = 'button';
</script>

<button class="button" {type}><slot /></button>

<style>
  .button {
    background: #ff3e00;
    color: #fff;
    padding: 0.5em 1em;
    border-radius: 3px;
    border: none;
    font-size: 1.2em;
    font-weight: 100;
    cursor: pointer;
    transition: all 0.2s ease-in-out;
  }
</style>
