# -*- coding: utf-8 -*-
"""
Caliopen mail message format management.

mail parsing is included in python, so this is not
getting external dependencies.

For formats with needs of external packages, they
must be defined outside of this one.
"""

import logging
import base64
import random
from itertools import groupby
from mailbox import Message
from datetime import datetime
from email.utils import parsedate_tz, mktime_tz

from caliopen_main.message.parameters import Participant, NewMessage
from caliopen_main.message.parameters import Attachment, ExternalReferences
from caliopen_main.user.helpers.normalize import clean_email_address
from .base import BaseRawParser, BaseAttachment


log = logging.getLogger(__name__)


class MailPart(BaseAttachment):
    """Mail part structure."""

    def __init__(self, part):
        """Initialize part structure."""
        self.content_type = part.get_content_type()
        self.filename = part.get_filename()
        text = part.get_payload()
        self.size = len(text) if text else 0
        self.can_index = False
        if 'text' in part.get_content_type():
            self.can_index = True
            charsets = part.get_charsets()
            if len(charsets) > 1:
                raise Exception('Too many charset %r for %s' %
                                (charsets, part.get_payload()))
            self.charset = charsets[0]
            if 'Content-Transfer-Encoding' in part.keys():
                if part.get('Content-Transfer-Encoding') == 'base64':
                    text = base64.b64decode(text)
            if self.charset:
                text = text.decode(self.charset, 'replace'). \
                    encode('utf-8')
        self.data = text


class MailMessage(BaseRawParser):
    """
    Mail message structure.

    Got a mail in raw rfc2822 format, parse it to
    resolve all recipients emails, parts and group headers
    """

    recipient_headers = ['From', 'To', 'Cc', 'Bcc']
    message_type = 'email'

    def __init__(self, raw_data):
        """Initialize structure from a raw mail."""
        super(MailMessage, self).__init__(raw_data)
        try:
            self.mail = Message(raw_data)
        except Exception as exc:
            log.error('Parse message failed %s' % exc)
            raise
        if self.mail.defects:
            # XXX what to do ?
            log.warn('Defects on parsed mail %r' % self.mail.defects)
            self.warning = self.mail.defects
        self.participants = self._get_participants()
        self.attachments = self._get_attachments()
        self.headers = self._get_headers()
        self.extra_parameters = self._get_extra_parameters()
        self.date = self._get_date()
        self.references = self._get_references()
        self.size = self._get_size()
        log.debug('Parsed mail {} with size {}'.
                  format(self.reference['message_id'], self.size))

    def _get_size(self):
        """Get mail size in octet."""
        return len(self.mail.as_string())

    def _get_references(self):
        """Return mail references to be used as ExternalReferences."""
        ext_id = self.mail.get('Message-Id')
        parent_id = self.mail.get('In-Reply-To')
        return {'message_id': ext_id,
                'parent_id': parent_id}

    def _get_date(self):
        """Get date from a mail message."""
        mail_date = self.mail.get('Date')
        if mail_date:
            tmp_date = parsedate_tz(mail_date)
            return datetime.fromtimestamp(mktime_tz(tmp_date))
        log.debug('No date on mail using now (UTC)')
        return datetime.utcnow()

    def _get_extra_parameters(self):
        """Return mail extra parameters informations."""
        return {'subject': self.mail.get('Subject')}

    def _get_participants(self):
        participants = []
        for header in self.recipient_headers:
            addrs = []
            participant_type = header.capitalize()
            if self.mail.get(header):
                if ',' in self.mail.get(header):
                    addrs.extend(self.mail.get(header).split(','))
                else:
                    addrs.append(self.mail.get(header))
            addrs = [clean_email_address(x) for x in addrs]
            for addr in addrs:
                participant = Participant()
                participant.address = addr[0]
                participant.type = participant_type
                participant.protocol = self.message_type
                participant.label = addr[1]
                participants.append(participant)
        return participants

    def _get_attachments(self):
        """Multipart message, extract parts."""
        parts = []
        for p in self.mail.walk():
            if not p.is_multipart():
                parts.append(self.__process_part(p))
        return parts

    def _get_headers(self):
        """
        Extract all headers into list.

        Duplicate on headers exists, group them by name
        with a related list of values
        """
        def keyfunc(item):
            return item[0]

        # Group multiple value for same headers into a dict of list
        headers = {}
        data = sorted(self.mail.items(), key=keyfunc)
        for k, g in groupby(data, key=keyfunc):
            headers[k] = [x[1] for x in g]
        return headers

    def __process_part(self, part):
        return MailPart(part)

    @property
    def importance_level(self):
        """Return percent estimated importance level of this message."""
        # XXX. real compute needed
        return 0 if self.spam_level else random.randint(50, 100)

    @property
    def lists(self):
        """List related to message."""
        lists = []
        for list_name in self.headers.get('List-ID', []):
            lists.append(list_name)
        return lists

    @property
    def from_(self):
        """Return the sender participant."""
        for part in self.participants:
            if part.type == 'From':
                return part
        return None

    def lookup_sequence(self):
        """Build parameter sequence for lookups."""
        seq = []
        # first from parent
        if self.external_parent_id:
            seq.append(('parent', self.external_parent_id))
        # then list lookup
        for listname in self.lists:
            seq.append(('list', listname))
        # last try to lookup from sender address
        if self.from_:
            seq.append(('from', self.from_.address))
        return seq

    def parse(self):
        """Transform mail to a NewMessage parameter."""
        msg = NewMessage()
        msg.type = self.message_type
        msg.subject = self.subject
        msg.date = self.date
        msg.body = self.text
        msg.is_unread = True
        msg.is_draft = False
        msg.is_answered = False
        msg.participants = self.participants
        msg.raw_msg_id = self.raw.raw_msg_id

        # XXX need transform to part parameter
        for part in self.attachments:
            param = Attachment()
            param.content_type = part.content_type
            param.data = part.data
            param.size = part.size
            param.filename = part.filename
            param.can_index = part.can_index
            msg.attachments.append(param)
        ext_ref = ExternalReferences()
        ext_ref.parent_id = self.external_parent_id
        ext_ref.message_id = self.external_message_id
        msg.external_references = ext_ref
        msg.importance_level = self.importance_level
        msg.privacy_features = self._get_privacy_features()
        return msg

    def _get_privacy_features(self):
        """Get privacy features from a mail message."""
        features = {}
        return features
