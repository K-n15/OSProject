<script setup>
const steps = ['Make a ticket', 'Wait for IT to accept', 'Check response']
const more = [
  { q: 'How do I know IT has accepted my ticket?', a: 'The status changes from pending to ongoing and the bell shows an update.' },
  { q: 'Can I edit a ticket after sending it?', a: 'Not yet. Open a new ticket and mention the earlier one.' },
  { q: 'What counts as closed?', a: 'IT marks a ticket closed once the issue is resolved. Closed tickets stay in your list.' },
]
</script>

<template>
  <div class="faq">
    <h2>How to use IT-Assist</h2>
    <ol class="steps"><li v-for="s in steps" :key="s">{{ s }}</li></ol>
    <h3>More questions</h3>
    <details v-for="m in more" :key="m.q"><summary>{{ m.q }}</summary><p>{{ m.a }}</p></details>
  </div>
</template>
