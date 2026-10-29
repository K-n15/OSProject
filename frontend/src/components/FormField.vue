<script setup>
import { useId } from 'vue'

const model = defineModel({ type: String, default: '' })
defineProps({
  label: { type: String, required: true },
  type: { type: String, default: 'text' },
  error: { type: String, default: '' },
  hint: { type: String, default: '' },
  autocomplete: { type: String, default: 'off' },
  inputmode: { type: String, default: undefined },
  maxlength: { type: Number, default: undefined },
  code: { type: Boolean, default: false },
  autofocus: { type: Boolean, default: false },
})
const id = useId()
</script>

<template>
  <div class="field">
    <label :for="id">{{ label }}</label>
    <input
      :id="id" v-model="model" :type="type" :autocomplete="autocomplete" :inputmode="inputmode"
      :maxlength="maxlength" :class="{ code }" :autofocus="autofocus"
      :aria-invalid="error ? 'true' : undefined"
      :aria-describedby="error ? id + '-err' : hint ? id + '-hint' : undefined"
    />
    <small v-if="error" :id="id + '-err'" class="field-error">{{ error }}</small>
    <small v-else-if="hint" :id="id + '-hint'" class="field-hint">{{ hint }}</small>
  </div>
</template>
