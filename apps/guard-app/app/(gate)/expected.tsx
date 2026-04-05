import { Text, YStack, ScrollView } from 'tamagui';

export default function ExpectedScreen() {
  return (
    <YStack flex={1} backgroundColor="$background" padding="$4">
      <Text fontSize="$6" fontWeight="bold" marginBottom="$4">
        Expected Today
      </Text>
      <ScrollView>
        {/* TODO: render DomesticHelp records expected today + pre-approved VisitorPasses */}
        <YStack alignItems="center" marginTop="$8">
          <Text color="$color10">No one expected right now</Text>
        </YStack>
      </ScrollView>
    </YStack>
  );
}
