import { useState } from 'react'
import {
  Avatar,
  Button,
  ButtonLink,
  Card,
  Dropdown,
  EmptyState,
  IconBadge,
  Input,
  ListRow,
  Modal,
  SearchBar,
  SectionHeader,
  Skeleton,
  Spinner,
  TagPill,
  Textarea,
} from '../components'

/**
 * Gallery is a development-only page that renders every base component in its
 * states so the design system can be checked visually (T3.3 verification).
 */
export function Gallery() {
  const [query, setQuery] = useState('')
  const [modalOpen, setModalOpen] = useState(false)

  return (
    <div className="placeholder">
      <SectionHeader title="Component gallery" headingLevel="h2" />

      <SectionHeader title="Buttons" />
      <div className="row" style={{ flexWrap: 'wrap' }}>
        <Button>Accent</Button>
        <Button variant="secondary">Secondary</Button>
        <Button variant="ghost">Ghost</Button>
        <Button variant="danger">Danger</Button>
        <Button loading>Loading</Button>
        <Button disabled>Disabled</Button>
        <Button size="sm">Small</Button>
        <Button size="lg">Large</Button>
        <ButtonLink to="/">Link button</ButtonLink>
      </div>

      <SectionHeader title="Cards & tags" />
      <div className="row" style={{ alignItems: 'stretch', flexWrap: 'wrap' }}>
        <Card style={{ maxWidth: 240 }}>
          <IconBadge label="Book">
            <span aria-hidden="true">📘</span>
          </IconBadge>
          <h4 style={{ marginTop: '1rem' }}>Workflows &amp; Guides</h4>
          <p className="text-secondary">Step-by-step procedures.</p>
          <Button block>Explore</Button>
        </Card>
        <Card interactive style={{ maxWidth: 240 }}>
          <p>Interactive card</p>
        </Card>
      </div>
      <div className="row" style={{ flexWrap: 'wrap' }}>
        <TagPill>Git</TagPill>
        <TagPill count={12}>React</TagPill>
        <TagPill active count={3}>
          Active
        </TagPill>
        <TagPill onClick={() => undefined} count={5}>
          Clickable
        </TagPill>
      </div>

      <SectionHeader title="Form controls" />
      <Input label="Title" placeholder="Document title" hint="Shown in lists" />
      <Input label="Slug" placeholder="slug" error="Slug already exists" />
      <Textarea label="Content" placeholder="Write markdown..." />
      <SearchBar value={query} onChange={setQuery} onSubmit={() => undefined} />

      <SectionHeader title="Rows & states" />
      <ListRow
        icon={<IconBadge round>📄</IconBadge>}
        title="Frontend Guide"
        meta="2h ago"
        action="View"
      />
      <ListRow title="Another doc" meta="Yesterday" action="View" />
      <EmptyState
        title="Nothing here yet"
        description="Create your first document."
        action={<Button>New</Button>}
      />
      <div className="row">
        <Spinner />
        <Spinner size="lg" />
        <Skeleton width={160} height={16} />
      </div>

      <SectionHeader title="Overlays" />
      <div className="row">
        <Button onClick={() => setModalOpen(true)}>Open modal</Button>
        <Dropdown
          items={[
            { label: 'Settings', onSelect: () => undefined },
            { label: 'Lock vault', onSelect: () => undefined },
          ]}
          trigger={({ toggle }) => (
            <Button variant="secondary" onClick={toggle}>
              Menu
            </Button>
          )}
        />
        <Avatar initials="AK" label="Alex K." />
      </div>

      <Modal
        open={modalOpen}
        title="Example modal"
        onClose={() => setModalOpen(false)}
        actions={
          <>
            <Button variant="secondary" onClick={() => setModalOpen(false)}>
              Cancel
            </Button>
            <Button onClick={() => setModalOpen(false)}>Confirm</Button>
          </>
        }
      >
        <p>Modals trap focus and close on Escape.</p>
      </Modal>
    </div>
  )
}
