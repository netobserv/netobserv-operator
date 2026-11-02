#!/usr/bin/env python3
"""Render the architecture Markdown and native diagrams into a Google Docs-ready DOCX.

Requires Pillow and fontconfig. Run from any directory; no network access needed.
PDF preview: libreoffice --headless --convert-to pdf --outdir docs/exports <docx>.
"""
from pathlib import Path
import html
import math
import re
import subprocess
import zipfile

from PIL import Image, ImageDraw, ImageFont

ROOT = Path(__file__).resolve().parents[1]
ASSETS = ROOT / 'docs/assets/haproxy-access-logs'
OUT = ROOT / 'docs/exports'
ASSETS.mkdir(parents=True, exist_ok=True)
OUT.mkdir(parents=True, exist_ok=True)
FONT = subprocess.check_output(['fc-match', '-f', '%{file}', 'Liberation Sans']).decode()
BOLD = subprocess.check_output(['fc-match', '-f', '%{file}', 'Liberation Sans:style=Bold']).decode()
NAVY, BLUE, TEAL, GRAY = '#17324D', '#245DB2', '#147D78', '#536578'


class Figure:
    def __init__(self, name, width, height):
        self.name, self.w, self.h = name, width, height
        self.im = Image.new('RGB', (width, height), 'white')
        self.draw = ImageDraw.Draw(self.im)
        self.svg = [f'<svg xmlns="http://www.w3.org/2000/svg" width="{width}" height="{height}" viewBox="0 0 {width} {height}"><rect width="100%" height="100%" fill="white"/>']

    def rect(self, box, fill, stroke=None, radius=18):
        self.draw.rounded_rectangle(box, radius=radius, fill=fill, outline=stroke, width=3)
        x, y, x2, y2 = box
        self.svg.append(f'<rect x="{x}" y="{y}" width="{x2-x}" height="{y2-y}" rx="{radius}" fill="{fill}" stroke="{stroke or fill}" stroke-width="3"/>')

    def text(self, x, y, text, size=30, color=NAVY, bold=False, center=False):
        font = ImageFont.truetype(BOLD if bold else FONT, size)
        bbox = self.draw.textbbox((0, 0), text, font=font)
        tx = x-(bbox[2]-bbox[0])/2 if center else x
        self.draw.text((tx, y-bbox[1]), text, font=font, fill=color)
        self.svg.append(f'<text x="{x}" y="{y+size*.78}" font-family="Arial, sans-serif" font-size="{size}" font-weight="{"bold" if bold else "normal"}" fill="{color}" text-anchor="{"middle" if center else "start"}">{html.escape(text)}</text>')

    def arrow(self, points, color=GRAY, both=False):
        self.draw.line(points, fill=color, width=5, joint='curve')
        self.svg.append(f'<polyline points="{" ".join(f"{x},{y}" for x,y in points)}" fill="none" stroke="{color}" stroke-width="5"/>')
        ends = [(points[-2], points[-1])]
        if both:
            ends.append((points[1], points[0]))
        for (x0,y0),(x,y) in ends:
            a = math.atan2(y-y0,x-x0)
            tri = [(x,y), (x-19*math.cos(a-.45),y-19*math.sin(a-.45)), (x-19*math.cos(a+.45),y-19*math.sin(a+.45))]
            self.draw.polygon(tri,fill=color)
            self.svg.append(f'<polygon points="{" ".join(f"{u},{v}" for u,v in tri)}" fill="{color}"/>')

    def box(self, x, y, width, title, subtitle='', color=BLUE, fill='#EEF4FC', height=116):
        self.rect((x,y,x+width,y+height),fill,color)
        self.text(x+width/2,y+25,title,34,color,True,True)
        if subtitle:
            self.text(x+width/2,y+72,subtitle,25,GRAY,center=True)

    def save(self):
        self.im.save(ASSETS / (self.name+'.png'),dpi=(300,300))
        (ASSETS/(self.name+'.svg')).write_text('\n'.join(self.svg)+ '\n</svg>\n')


def diagrams():
    f = Figure('architecture',1800,950)
    f.text(55,28,'APPLICATION TRAFFIC',25,GRAY,True)
    for x,title,sub in [(55,'Client','Reported source identity'),(490,'External LB','Preserve source / PROXY'),(925,'HAProxy router','OpenShift ingress'),(1360,'Application','Backend endpoint')]:
        f.box(x,100,365,title,sub,color=NAVY,fill='#F2F5F8')
    for a,b in [(420,490),(855,925),(1290,1360)]:
        f.arrow([(a+8,158),(b-10,158)],NAVY)
    f.text(55,294,'OBSERVABILITY DATA',25,GRAY,True)
    f.box(925,365,365,'FLP proxy input','Validate + enrich requests',color=TEAL,fill='#EDF8F5')
    f.arrow([(1107,222),(1107,355)],TEAL)
    f.text(1140,274,'UDP syslog',27,TEAL)
    f.box(1360,365,365,'eBPF agent','Ordinary flow capture')
    f.arrow([(1320,164),(1320,265),(1542,265),(1542,355)],BLUE)
    f.box(1360,555,365,'FLP flow pipeline','Existing flow processing')
    f.arrow([(1542,487),(1542,545)],BLUE)
    f.box(925,755,365,'Loki','Separate record streams',color=NAVY,fill='#F2F5F8',height=130)
    f.arrow([(1107,487),(1107,745)],TEAL)
    f.text(1140,617,'Proxy records',27,TEAL)
    f.arrow([(1542,677),(1542,820),(1300,820)],BLUE)
    f.text(1435,720,'Flow records',27,BLUE,center=True)
    f.box(55,755,630,'Console + authorized backend','Related clients → Find related clients',height=130)
    f.arrow([(700,820),(910,820)],NAVY,both=True)
    f.text(803,767,'Lookup',27,NAVY,center=True)
    f.text(55,402,'Independent evidence',35,NAVY,True)
    f.text(55,460,'One proxy record per completed request.',29,GRAY)
    f.text(55,507,'Flow capture continues unchanged.',29,GRAY)
    f.text(55,554,'Correlation happens when a user asks.',29,GRAY)
    f.save()
    f = Figure('timing',1800,480)
    f.rect((120,90,1720,375),'#F2F5F8',radius=12)
    f.text(148,108,'RECEPTION-TIME SEARCH WINDOW',25,GRAY,True)
    f.rect((200,180,670,225),'#245DB2',radius=8)
    f.text(200,143,'Observed flow interval',27,BLUE,True)
    f.arrow([(670,231),(670,260)],BLUE)
    f.text(680,246,'Flow export',24,BLUE)
    f.rect((270,297,1230,342),'#147D78',radius=8)
    f.text(280,260,'HTTP request interval',27,TEAL,True)
    f.arrow([(1240,320),(1390,320)],TEAL)
    f.text(1410,299,'Log received',27,TEAL,True)
    f.text(1080,350,'Request completes',24,TEAL)
    f.text(130,405,'Flow start − 2 s',25,GRAY)
    f.text(1000,405,'Flow end + 10 min (or now)',25,GRAY)
    f.save()


W='http://schemas.openxmlformats.org/wordprocessingml/2006/main'
R='http://schemas.openxmlformats.org/officeDocument/2006/relationships'
REL='http://schemas.openxmlformats.org/package/2006/relationships'
A='http://schemas.openxmlformats.org/drawingml/2006/main'
WP='http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing'
PIC='http://schemas.openxmlformats.org/drawingml/2006/picture'
E=lambda s:html.escape(str(s),quote=True)


class Word:
    def __init__(self):
        self.body=[]
        self.rels=[('rId1','styles','styles.xml'),('rId2','header','header1.xml'),('rId3','footer','footer1.xml')]
        self.media=[]

    def run(self,text,bold=False,italic=False,mono=False):
        props=('<w:b/>'if bold else '')+('<w:i/>'if italic else '')+('<w:rFonts w:ascii="Courier New" w:hAnsi="Courier New"/><w:sz w:val="18"/>'if mono else '')
        return f'<w:r><w:rPr>{props}</w:rPr><w:t xml:space="preserve">{E(text)}</w:t></w:r>'

    def inline(self,text):
        parts=[]
        pattern=r'(\[[^\]]+\]\(https?://[^)]+\)|`[^`]+`|\*\*[^*]+\*\*|\*[^*]+\*)'
        for item in re.split(pattern,text):
            if not item:continue
            if item.startswith('['):
                m=re.fullmatch(r'\[([^\]]+)\]\(([^)]+)\)',item)
                rid=f'rId{len(self.rels)+1}';self.rels.append((rid,'hyperlink',m[2]))
                parts.append(f'<w:hyperlink r:id="{rid}"><w:r><w:rPr><w:color w:val="245DB2"/><w:u w:val="single"/></w:rPr><w:t>{E(m[1])}</w:t></w:r></w:hyperlink>')
            elif item.startswith('`'):parts.append(self.run(item[1:-1],mono=True))
            elif item.startswith('**'):parts.append(self.run(item[2:-2],bold=True))
            elif item.startswith('*'):parts.append(self.run(item[1:-1],italic=True))
            else:parts.append(self.run(item))
        return ''.join(parts)

    def para(self,text='',style='Normal',keep=False):
        self.body.append(f'<w:p><w:pPr><w:pStyle w:val="{style}"/>{"<w:keepNext/>"if keep else ""}</w:pPr>{self.inline(text)}</w:p>')

    def table(self,rows):
        n=len(rows[0]);ratios=[.32,.68]if n==2 else [.33,.22,.45]
        if rows[0][0]=='Deployment model':ratios=[.18,.45,.37]
        if rows[0][0]=='End-to-end check':ratios=[.64,.18,.18]
        widths=[round(9826*x)for x in ratios]
        xml=['<w:tbl><w:tblPr><w:tblW w:w="9826" w:type="dxa"/><w:tblLayout w:type="fixed"/><w:tblCellMar><w:top w:w="85" w:type="dxa"/><w:left w:w="100" w:type="dxa"/><w:bottom w:w="85" w:type="dxa"/><w:right w:w="100" w:type="dxa"/></w:tblCellMar><w:tblBorders><w:bottom w:val="single" w:sz="4" w:color="D6DFE8"/><w:insideH w:val="single" w:sz="4" w:color="D6DFE8"/></w:tblBorders></w:tblPr><w:tblGrid>']
        xml.extend(f'<w:gridCol w:w="{v}"/>'for v in widths);xml.append('</w:tblGrid>')
        for i,row in enumerate(rows):
            xml.append('<w:tr><w:trPr><w:cantSplit/>'+('<w:tblHeader/>'if i==0 else '')+'</w:trPr>')
            for width,value in zip(widths,row):
                shade='17324D'if i==0 else('F2F5F8'if i%2 else 'FFFFFF')
                content=self.inline(value)
                if i==0:
                    content=f'<w:r><w:rPr><w:b/><w:color w:val="FFFFFF"/></w:rPr><w:t>{E(value)}</w:t></w:r>'
                xml.append(f'<w:tc><w:tcPr><w:tcW w:w="{width}" w:type="dxa"/><w:shd w:fill="{shade}"/><w:vAlign w:val="center"/></w:tcPr><w:p><w:pPr><w:pStyle w:val="TableText"/></w:pPr>{content}</w:p></w:tc>')
            xml.append('</w:tr>')
        xml.append('</w:tbl><w:p><w:pPr><w:spacing w:after="20"/></w:pPr></w:p>');self.body.extend(xml)

    def image(self,alt,path):
        file=ROOT/'docs'/path
        with Image.open(file)as im:width,height=im.size
        cx=6240000;cy=round(cx*height/width)
        rid=f'rId{len(self.rels)+1}';name=f'image{len(self.media)+1}.png'
        self.rels.append((rid,'image','media/'+name));self.media.append((name,file))
        self.body.append(f'''<w:p><w:pPr><w:keepNext/><w:spacing w:after="70"/></w:pPr><w:r><w:drawing><wp:inline distT="0" distB="0" distL="0" distR="0"><wp:extent cx="{cx}" cy="{cy}"/><wp:docPr id="{len(self.media)}" name="{name}" descr="{E(alt)}"/><a:graphic><a:graphicData uri="{PIC}"><pic:pic><pic:nvPicPr><pic:cNvPr id="0" name="{name}"/><pic:cNvPicPr/></pic:nvPicPr><pic:blipFill><a:blip r:embed="{rid}"/><a:stretch><a:fillRect/></a:stretch></pic:blipFill><pic:spPr><a:xfrm><a:off x="0" y="0"/><a:ext cx="{cx}" cy="{cy}"/></a:xfrm><a:prstGeom prst="rect"><a:avLst/></a:prstGeom></pic:spPr></pic:pic></a:graphicData></a:graphic></wp:inline></w:drawing></w:r></w:p>''')

    def render(self,text):
        lines=text.splitlines();i=0
        while i<len(lines):
            line=lines[i].strip()
            if not line:i+=1;continue
            if line=='<!-- pagebreak -->':self.body.append('<w:p><w:r><w:br w:type="page"/></w:r></w:p>');i+=1;continue
            if line.startswith('!['):
                m=re.fullmatch(r'!\[(.*?)\]\((.*?)\)',line);self.image(m[1],m[2]);i+=1;continue
            if line.startswith('```'):
                i+=1
                while not lines[i].startswith('```'):
                    self.para(lines[i],'Code',keep=not lines[i+1].startswith('```'));i+=1
                i+=1;continue
            if line.startswith('|'):
                rows=[]
                while i<len(lines)and lines[i].startswith('|'):
                    values=[v.strip()for v in lines[i].strip('|').split('|')]
                    if not all(re.fullmatch(r':?-+:?',v)for v in values):rows.append(values)
                    i+=1
                self.table(rows);continue
            if line.startswith('#'):
                level=len(line)-len(line.lstrip('#'));style={1:'Title',2:'Subtitle',3:'Heading1',4:'Heading2'}[level]
                self.para(line[level:].strip(),style,keep=True);i+=1;continue
            style='Normal'
            if line.startswith('*Figure'):style='Caption'
            if line.startswith('- '):line='• '+line[2:];style='ListText'
            elif re.match(r'^\d\. ',line):style='ListText'
            if line.startswith('**Design review'):style='Metadata'
            self.para(line,style);i+=1

    def save(self):
        sect='<w:sectPr><w:headerReference w:type="default" r:id="rId2"/><w:footerReference w:type="default" r:id="rId3"/><w:pgSz w:w="11906" w:h="16838"/><w:pgMar w:top="930" w:right="1040" w:bottom="930" w:left="1040" w:header="400" w:footer="400"/></w:sectPr>'
        doc=f'<w:document xmlns:w="{W}" xmlns:r="{R}" xmlns:a="{A}" xmlns:wp="{WP}" xmlns:pic="{PIC}"><w:body>'+''.join(self.body)+sect+'</w:body></w:document>'
        styles=f'''<w:styles xmlns:w="{W}"><w:docDefaults><w:rPrDefault><w:rPr><w:rFonts w:ascii="Arial" w:hAnsi="Arial"/><w:sz w:val="21"/><w:color w:val="25364A"/><w:lang w:val="en-US"/></w:rPr></w:rPrDefault><w:pPrDefault><w:pPr><w:spacing w:after="115" w:line="258" w:lineRule="auto"/><w:widowControl/></w:pPr></w:pPrDefault></w:docDefaults>'''
        for name,size,color,after,before,bold in [('Normal',21,'25364A',115,0,False),('Title',48,'17324D',100,0,True),('Subtitle',26,'536578',150,0,False),('Metadata',18,'536578',180,0,False),('Heading1',30,'17324D',130,80,True),('Heading2',23,'147D78',90,130,True),('Caption',17,'536578',100,0,False),('TableText',19,'25364A',0,0,False),('Code',19,'25364A',0,0,False),('ListText',21,'25364A',65,0,False)]:
            outline='<w:keepNext/><w:outlineLvl w:val="'+('0'if name=='Heading1'else '1')+'"/>'if name in('Heading1','Heading2')else ''
            code='<w:shd w:fill="F2F5F8"/><w:ind w:left="130"/>'if name=='Code'else ''
            rfont='<w:rFonts w:ascii="Courier New" w:hAnsi="Courier New"/>'if name=='Code'else ''
            base='<w:basedOn w:val="Normal"/>'if name!='Normal'else ''
            styles+=f'<w:style w:type="paragraph" w:styleId="{name}"><w:name w:val="{name}"/>{base}<w:pPr>{outline}{code}<w:spacing w:before="{before}" w:after="{after}"/></w:pPr><w:rPr>{rfont}<w:sz w:val="{size}"/><w:color w:val="{color}"/>{"<w:b/>"if bold else ""}</w:rPr></w:style>'
        styles+='</w:styles>'
        header=f'<w:hdr xmlns:w="{W}"><w:p><w:pPr><w:spacing w:after="0"/></w:pPr><w:r><w:rPr><w:sz w:val="16"/><w:color w:val="536578"/></w:rPr><w:t>NETOBSERV   /   HAProxy access-log correlation</w:t></w:r></w:p></w:hdr>'
        footer=f'<w:ftr xmlns:w="{W}"><w:p><w:pPr><w:jc w:val="right"/></w:pPr><w:r><w:rPr><w:sz w:val="16"/><w:color w:val="536578"/></w:rPr><w:t>Architecture · 29 October 2026     |     </w:t></w:r><w:fldSimple w:instr="PAGE"/></w:p></w:ftr>'
        rels=f'<Relationships xmlns="{REL}">'
        for rid,kind,target in self.rels:
            mode=' TargetMode="External"'if kind=='hyperlink'else ''
            rels+=f'<Relationship Id="{rid}" Type="{R}/{kind}" Target="{E(target)}"{mode}/>'
        rels+='</Relationships>'
        ct='<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Default Extension="png" ContentType="image/png"/>'
        for part,kind in [('document','document.main'),('styles','styles'),('header1','header'),('footer1','footer')]:ct+=f'<Override PartName="/word/{part}.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.{kind}+xml"/>'
        ct+='</Types>'
        path=OUT/'HAProxy-Access-Log-Correlation-Architecture.docx'
        with zipfile.ZipFile(path,'w',zipfile.ZIP_DEFLATED)as z:
            z.writestr('[Content_Types].xml',ct)
            z.writestr('_rels/.rels',f'<Relationships xmlns="{REL}"><Relationship Id="rId1" Type="{R}/officeDocument" Target="word/document.xml"/></Relationships>')
            for name,xml in [('document',doc),('styles',styles),('header1',header),('footer1',footer)]:z.writestr('word/'+name+'.xml','<?xml version="1.0" encoding="UTF-8" standalone="yes"?>'+xml)
            z.writestr('word/_rels/document.xml.rels',rels)
            for name,file in self.media:z.write(file,'word/media/'+name)
        print(path)


if __name__=='__main__':
    diagrams()
    word=Word();word.render((ROOT/'docs/HaproxyAccessLogCorrelation.md').read_text());word.save()
