import { screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'

/** Opens a row's actions menu (by its trigger's accessible name) and returns the trigger. */
export async function openRowMenu(triggerName: string | RegExp) {
  const trigger = await screen.findByRole('button', { name: triggerName })
  await userEvent.click(trigger)
  return trigger
}

/** Opens a row's actions menu and clicks the named item. */
export async function pickRowAction(triggerName: string | RegExp, item: string | RegExp) {
  const trigger = await openRowMenu(triggerName)
  await userEvent.click(await screen.findByRole('menuitem', { name: item }))
  return trigger
}
