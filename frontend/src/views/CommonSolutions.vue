<script setup>
import { ref, computed } from 'vue'
import { solutions } from '../data/solutions'
const categories = Object.keys(solutions)
const active = ref(categories[0])
const articles = computed(() => solutions[active.value])
</script>

<template>
  <div class="split">
    <ul class="category-list">
      <li v-for="c in categories" :key="c">
        <button :class="{ active: c === active }" @click="active = c">{{ c }}</button>
      </li>
    </ul>
    <section class="articles">
      <h2>{{ active }}</h2>
      <article v-for="a in articles" :key="a.title">
        <h3>{{ a.title }}</h3>
        <p>{{ a.body }}</p>
      </article>
    </section>
  </div>
</template>
