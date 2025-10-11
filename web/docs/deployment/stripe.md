# 💵 Stripe
DeploySolo comes with Stripe Checkout and Webhook integration. This should be given proper care, consideration, and testing. This documentation is not a full Stripe guide, but is an introductory overview.

A product needs to be created, and its **Price ID** needs to be retrieved. 


## Create a Product

![0](/public/images/doc/stripe/stripe-0.png)

![1](/public/images/doc/stripe/stripe-1.png)

![2](/public/images/doc/stripe/stripe-2.png)

Make a note of the **Price ID**. This will be used in the .env file, not the Product ID.

## Create Webhook

![3](/public/images/doc/stripe/stripe-3.png)

## Testing Locally
The stripe webhook can be tested locally with the [Stripe CLI](https://github.com/stripe/stripe-cli/wiki/listen-command). After installing it on the EC2 instance, you can start a listener with:

```sh
stripe listen --forward-to localhost:8090/webhook
```

To forward a simulated successful completed checkout for a given email, the following command can be used:
```sh
stripe trigger checkout.session.completed --add checkout_session:metadata.email=user@email.com
```

## Update .env

Update `.env` accordingly.

```
STRIPE_SECRET_KEY=sk_live_...
STRIPE_WEBHOOK_SECRET=whsec_...
STRIPE_PRICE_ID=price_...
```
