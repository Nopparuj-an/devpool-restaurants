import { notFound } from "next/navigation";

import { ScreenView } from "../../registry";
import { isScreen } from "../../screen-list";

// One screen at full size. The /design gallery shows these in frames, and
// each one can also be opened on its own.
export default async function DesignScreen({ params }: PageProps<"/design/screens/[screen]">) {
  const { screen } = await params;
  if (!isScreen(screen)) notFound();
  return <ScreenView name={screen} />;
}
