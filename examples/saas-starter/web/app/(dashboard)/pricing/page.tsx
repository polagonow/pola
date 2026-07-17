import { Check } from "lucide-react";
import { PaymentsAction, type PriceView, type ProductView } from "@pola/actions";
import { SubmitButton } from "./submit-button";

const FEATURES: Record<string, string[]> = {
  Base: ["Unlimited Usage", "Unlimited Workspace Members", "Email Support"],
  Plus: [
    "Everything in Base, and:",
    "Early Access to New Features",
    "24/7 Support + Slack Access",
  ],
};

const DEFAULT_FEATURES = [
  "Unlimited Usage",
  "Unlimited Workspace Members",
  "Email Support",
];

// Intl is unavailable in the Goja SSR engine, so format currency by hand.
function formatCurrency(amountCents: number, currency: string): string {
  const value = (amountCents / 100).toFixed(2);
  const code = (currency || "usd").toUpperCase();
  const symbols: Record<string, string> = { USD: "$", EUR: "€", GBP: "£" };
  const symbol = symbols[code] ?? "";
  return symbol ? `${symbol}${value}` : `${value} ${code}`;
}

export default async function PricingPage() {
  let prices: PriceView[] = [];
  let products: ProductView[] = [];
  try {
    [prices, products] = await Promise.all([
      PaymentsAction.prices(),
      PaymentsAction.products(),
    ]);
  } catch {
    // Stripe not configured or API error — fall through to the empty state
    // rather than crashing the server render.
  }

  if (!products.length) {
    return (
      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-12">
        <div className="max-w-xl mx-auto text-center py-16">
          <h1 className="text-2xl font-medium text-gray-900 mb-3">Pricing</h1>
          <p className="text-gray-600">
            No plans are available right now. Configure your Stripe keys to see
            live pricing here.
          </p>
        </div>
      </main>
    );
  }

  const priceFor = (product: ProductView): PriceView | undefined =>
    prices.find((p) => p.id === product.defaultPriceId) ||
    prices.find((p) => p.productId === product.id);

  return (
    <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-12">
      <div className="grid md:grid-cols-2 gap-8 max-w-xl mx-auto">
        {products.map((product) => {
          const price = priceFor(product);
          return (
            <PricingCard
              key={product.id}
              name={product.name}
              description={product.description}
              price={price?.unitAmount ?? 0}
              interval={price?.interval || "month"}
              currency={price?.currency || "usd"}
              features={FEATURES[product.name] || DEFAULT_FEATURES}
              priceId={price?.id}
            />
          );
        })}
      </div>
    </main>
  );
}

function PricingCard({
  name,
  description,
  price,
  interval,
  currency,
  features,
  priceId,
}: {
  name: string;
  description?: string;
  price: number;
  interval: string;
  currency: string;
  features: string[];
  priceId?: string;
}) {
  // NOTE: the Goja SSR engine does not implement the Intl API, so format the
  // currency manually instead of using Intl.NumberFormat.
  const formatted = formatCurrency(price, currency);

  return (
    <div className="pt-6">
      <h2 className="text-2xl font-medium text-gray-900 mb-2">{name}</h2>
      {description ? (
        <p className="text-sm text-gray-600 mb-4">{description}</p>
      ) : null}
      <p className="text-sm text-gray-600 mb-4">with 14 day free trial</p>
      <p className="text-4xl font-medium text-gray-900 mb-6">
        {formatted}{" "}
        <span className="text-xl font-normal text-gray-600">
          per user / {interval}
        </span>
      </p>
      <ul className="space-y-4 mb-8">
        {features.map((feature, index) => (
          <li key={index} className="flex items-start">
            <Check className="h-5 w-5 text-orange-500 mr-2 mt-0.5 flex-shrink-0" />
            <span className="text-gray-700">{feature}</span>
          </li>
        ))}
      </ul>
      <SubmitButton priceId={priceId} />
    </div>
  );
}
