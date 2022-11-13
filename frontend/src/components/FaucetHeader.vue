<template lang="pug">
.section-header
  h1 Secret Network Fee Grant Faucet

  p Use this faucet to get a fee grant for your Secret Network wallet.
  
  p A Fee Grant gives you enough SCRT (up to 0.1 SCRT) to pay for transactions to fund your wallet and can only be used for paying transaction fees. 

  p It automatically expires after 1 day, after which you can get a new fee grant.
  
  p To use the fee grant, please add the address secret1tq6y8waegggp4fv2fcxk3zmpsmlfadyc7lsd69 <button @click="copy">Copy</button> to the "granter" flag in your Fina Cash wallet.
  notifications(:notifications='notifications' theme='cosmos')
</template>

<script>
import { mapGetters } from "vuex";
import Notifications from "@nylira/vue-notifications";

export default {
  name: "faucet-header",
  computed: {
    ...mapGetters(["notifications","config"])
  },
  methods: {
    copy() {
        navigator.clipboard.writeText("secret1tq6y8waegggp4fv2fcxk3zmpsmlfadyc7lsd69");
        this.$store.commit("notify", {
            title: "Successfully Granted Fee",
            body:  `Copied fee granter address secret1tq6y8waegggp4fv2fcxk3zmpsmlfadyc7lsd69 to clipboard`
          });
    }
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
