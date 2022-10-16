<template lang="pug">
.section-header
  h1 Secret Network Fee Grant Faucet
  p Use this faucet to get a fee grant for your Secret Network wallet. 

  p A Fee Grant gives you enough SCRT (0.01 SCRT) to pay for a transaction to fund your wallet and can only be used for paying transaction fees.

  p It automatically expires after 1 day, after which you can get a new fee grant. 
</template>

<script>
import { mapGetters } from "vuex";
export default {
  name: "faucet-header",
  computed: {
    ...mapGetters(["config"])
  }
};
</script>

<style lang="stylus">
@import '~variables';

.section-header {
  padding: 0 1rem 1.5rem;

  span.chain {
    white-space: nowrap;
  }

  strong {
    font-weight: 500;
  }

  h1 {
    font-size: 1.5rem;
    font-weight: 600;
    color: var(--bright);
    line-height: 1.25;
  }

  h1 + p {
    margin-top: 0.75rem;
  }

  p {
    color: var(--dim);
  }
}
</style>
