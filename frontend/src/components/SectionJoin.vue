<template lang="pug">
.section
  .section-header
    h1 Don't have a Secret Network mainnet address?
    p There's two ways to get one. The first is by using Keplr, the crypto wallet for Secret. If you know command line, you can also generate an address with the secretcli.
</template>

<script>
import Btn from "@nylira/vue-button";
export default {
  name: "section-join",
  components: {
    Btn
  }
};
</script>
