import { Text, Heading, Section } from "@react-email/components";

interface InviteProps {
  // TODO: define props
}

export default function Invite({}: InviteProps) {
  return (
    <Section>
      <Heading>Invite</Heading>
      <Text>This is the invite email template.</Text>
    </Section>
  );
}
