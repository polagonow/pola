"use client";

import { useState, useTransition } from "react";
import { Button } from "@/components/ui/button";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  CardFooter,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { Label } from "@/components/ui/label";
import { Loader2, PlusCircle } from "lucide-react";
import { mutate } from "swr";
import { useTeam, useUser, type TeamMember } from "@/lib/hooks";
import {
  customerPortal,
  inviteTeamMember,
  removeTeamMember,
} from "@/lib/actions";

function getDisplayName(m: { name?: string; email?: string }) {
  return m.name || m.email || "Unknown User";
}

function ManageSubscription() {
  const { data: teamData } = useTeam();
  const [isPending, startTransition] = useTransition();

  function handlePortal() {
    startTransition(async () => {
      const res = await customerPortal();
      if (res?.redirect) window.location.href = res.redirect;
    });
  }

  return (
    <Card className="mb-8">
      <CardHeader>
        <CardTitle>Team Subscription</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="space-y-4">
          <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center">
            <div className="mb-4 sm:mb-0">
              <p className="font-medium">
                Current Plan: {teamData?.planName || "Free"}
              </p>
              <p className="text-sm text-muted-foreground">
                {teamData?.subscriptionStatus === "active"
                  ? "Billed monthly"
                  : teamData?.subscriptionStatus === "trialing"
                    ? "Trial period"
                    : "No active subscription"}
              </p>
            </div>
            <Button
              type="button"
              variant="outline"
              onClick={handlePortal}
              disabled={isPending}
            >
              {isPending ? "Loading..." : "Manage Subscription"}
            </Button>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}

function TeamMembers() {
  const { data: teamData } = useTeam();
  const { data: user } = useUser();
  const isOwner = user?.role === "owner";
  const [error, setError] = useState<string | null>(null);
  const [pendingId, setPendingId] = useState<number | null>(null);
  const [, startTransition] = useTransition();

  function handleRemove(member: TeamMember) {
    setError(null);
    setPendingId(member.id);
    startTransition(async () => {
      const res = await removeTeamMember(member.id);
      setPendingId(null);
      if (res?.error) {
        setError(res.error);
      } else {
        mutate("/api/team");
      }
    });
  }

  if (!teamData?.members?.length) {
    return (
      <Card className="mb-8">
        <CardHeader>
          <CardTitle>Team Members</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-muted-foreground">No team members yet.</p>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card className="mb-8">
      <CardHeader>
        <CardTitle>Team Members</CardTitle>
      </CardHeader>
      <CardContent>
        <ul className="space-y-4">
          {teamData.members.map((member, index) => (
            <li key={member.id} className="flex items-center justify-between">
              <div className="flex items-center space-x-4">
                <Avatar>
                  <AvatarFallback>
                    {getDisplayName(member)
                      .split(" ")
                      .map((n) => n[0])
                      .join("")}
                  </AvatarFallback>
                </Avatar>
                <div>
                  <p className="font-medium">{getDisplayName(member)}</p>
                  <p className="text-sm text-muted-foreground capitalize">
                    {member.role}
                  </p>
                </div>
              </div>
              {isOwner && index > 1 ? (
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  disabled={pendingId === member.id}
                  onClick={() => handleRemove(member)}
                >
                  {pendingId === member.id ? "Removing..." : "Remove"}
                </Button>
              ) : null}
            </li>
          ))}
        </ul>
        {error && <p className="text-red-500 mt-4">{error}</p>}
      </CardContent>
    </Card>
  );
}

function InviteTeamMember() {
  const { data: user } = useUser();
  const isOwner = user?.role === "owner";
  const [email, setEmail] = useState("");
  const [role, setRole] = useState("member");
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const [isPending, startTransition] = useTransition();

  function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setSuccess(null);
    startTransition(async () => {
      const res = await inviteTeamMember(email, role);
      if (res?.error) {
        setError(res.error);
      } else if (res?.success) {
        setSuccess(res.success);
        setEmail("");
      }
    });
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Invite Team Member</CardTitle>
      </CardHeader>
      <CardContent>
        <form onSubmit={onSubmit} className="space-y-4">
          <div>
            <Label htmlFor="invite-email" className="mb-2">
              Email
            </Label>
            <Input
              id="invite-email"
              name="email"
              type="email"
              placeholder="Enter email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              required
              disabled={!isOwner}
            />
          </div>
          <div>
            <Label>Role</Label>
            <RadioGroup
              value={role}
              onValueChange={setRole}
              name="role"
              className="flex space-x-4"
              disabled={!isOwner}
            >
              <div className="flex items-center space-x-2 mt-2">
                <RadioGroupItem value="member" id="member" />
                <Label htmlFor="member">Member</Label>
              </div>
              <div className="flex items-center space-x-2 mt-2">
                <RadioGroupItem value="owner" id="owner" />
                <Label htmlFor="owner">Owner</Label>
              </div>
            </RadioGroup>
          </div>
          {error && <p className="text-red-500">{error}</p>}
          {success && <p className="text-green-500">{success}</p>}
          <Button
            type="submit"
            className="bg-orange-500 hover:bg-orange-600 text-white"
            disabled={isPending || !isOwner}
          >
            {isPending ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                Inviting...
              </>
            ) : (
              <>
                <PlusCircle className="mr-2 h-4 w-4" />
                Invite Member
              </>
            )}
          </Button>
        </form>
      </CardContent>
      {!isOwner && (
        <CardFooter>
          <p className="text-sm text-muted-foreground">
            You must be a team owner to invite new members.
          </p>
        </CardFooter>
      )}
    </Card>
  );
}

export default function SettingsPage() {
  return (
    <section className="flex-1 p-4 lg:p-8">
      <h1 className="text-lg lg:text-2xl font-medium mb-6">Team Settings</h1>
      <ManageSubscription />
      <TeamMembers />
      <InviteTeamMember />
    </section>
  );
}
