import ContactForm from "../../components/ContactForm";
import { submitContact } from "../actions";

export default function ContactPage() {
  return (
    <div style={{ maxWidth: 600, margin: "0 auto" }}>
      <h1>Contact Us</h1>
      <p style={{ color: "var(--muted)", marginBottom: 24 }}>
        Send us a message using the form below. This form is powered by React
        Server Actions.
      </p>
      <ContactForm submitAction={submitContact} />
    </div>
  );
}
